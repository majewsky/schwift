/******************************************************************************
*
*  Copyright 2018 Stefan Majewsky <majewsky@gmx.net>
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
******************************************************************************/

package schwift

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/jpillora/longestcommon"
)

//SegmentInfo describes a segment of a large object.
//
//For .RangeLength == 0, the segment consists of all the bytes in the backing
//object, after skipping the first .RangeOffset bytes. The default
//(.RangeOffset == 0) includes the entire contents of the backing object.
//
//For .RangeLength > 0, the segment consists of that many bytes from the
//backing object, again after skipping the first .RangeOffset bytes.
//
//However, for .RangeOffset < 0, the segment consists of .RangeLength many bytes
//from the *end* of the backing object. (The concrete value for .RangeOffset is
//disregarded.) .RangeLength must be non-zero in this case.
//
//Sorry that specifying a range is that involved. I was just following orders ^W
//RFC 7233, section 3.1 here.
type SegmentInfo struct {
	Object      *Object
	SizeBytes   uint64
	Etag        string
	RangeLength uint64
	RangeOffset int64
	//Static Large Objects support data segments that are not backed by actual
	//objects. For those kinds of segments, only the Data attribute is set and
	//all other attributes are set to their default values (esp. .Object == nil).
	//
	//Data segments can only be used for small chunks of data because the SLO
	//manifest (the list of all SegmentInfo encoded as JSON) is severely limited
	//in size (usually to 8 MiB).
	Data []byte
}

type sloSegmentInfo struct {
	Path       string `json:"path,omitempty"`
	SizeBytes  uint64 `json:"size_bytes,omitempty"`
	Etag       string `json:"etag,omitempty"`
	Range      string `json:"range,omitempty"`
	DataBase64 string `json:"data,omitempty"`
}

//LargeObjectOpenMode is a set of flags that can be given to
//LargeObject.Open().
type LargeObjectOpenMode int

const (
	//OpenTruncate indicates that all existing segments in this object shall be
	//deleted by Open().
	OpenTruncate LargeObjectOpenMode = 0
	//OpenAppend indicates that Open() shall set up the writer to append new
	//content to the existing segments.
	OpenAppend LargeObjectOpenMode = 1 << 0
	//OpenKeepSegments indicates that, when truncating an existing object, the
	//segments shall not be deleted even though they are no longer referenced by
	//this object. This flag has no effect when combined with OpenAppend.
	OpenKeepSegments LargeObjectOpenMode = 1 << 1
)

//LargeObjectStrategy is an enum of segmenting strategies supported by Swift.
type LargeObjectStrategy int

const (
	//StaticLargeObject is the default LargeObjectStrategy used by Schwift.
	StaticLargeObject LargeObjectStrategy = iota
	//DynamicLargeObject is an older LargeObjectStrategy that is not recommended
	//for new applications because of eventual consistency problems and missing
	//support for several newer features (e.g. data segments, range specifications).
	DynamicLargeObject
)

////////////////////////////////////////////////////////////////////////////////

//LargeObject is a wrapper for type Object that performs operations specific to
//large objects.
//
//This type should only be constructed through the Object.AsLargeObject()
//method. If the object does not exist yet, the SegmentContainerName and
//SegmentPrefix must be specified before this object can be written to, and the
//Strategy can be adjusted in the unlikely case that an SLO is not desired.
type LargeObject struct {
	Object           *Object
	SegmentContainer *Container
	SegmentPrefix    string
	Strategy         LargeObjectStrategy
	//This is private so that we can later optimize this to load the segments
	//only on demand.
	segments []SegmentInfo
}

//AsLargeObject prepares a LargeObject instance. If the given object exists,
//but is not a large object, ErrNotLarge will be returned. If the given object
//does not yet exist, the SegmentContainer and SegmentPrefix attributes need to
//be filled in before the LargeObject can be used.
func (o *Object) AsLargeObject() (*LargeObject, error) {
	exists, err := o.Exists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return &LargeObject{Object: o, Strategy: StaticLargeObject}, nil
	}

	h := o.headers
	if h.IsDynamicLargeObject() {
		return o.asDLO(h.Get("X-Object-Manifest"))
	}
	if h.IsStaticLargeObject() {
		return o.asSLO()
	}
	return nil, ErrNotLarge
}

func (o *Object) asDLO(manifestStr string) (*LargeObject, error) {
	manifest := strings.SplitN(manifestStr, "/", 2)
	if len(manifest) < 2 {
		return nil, ErrNotLarge
	}

	lo := &LargeObject{
		Object:           o,
		SegmentContainer: o.c.a.Container(manifest[0]),
		SegmentPrefix:    manifest[1],
		Strategy:         DynamicLargeObject,
	}

	iter := lo.SegmentContainer.Objects()
	iter.Prefix = lo.SegmentPrefix
	segmentInfos, err := iter.CollectDetailed()
	if err != nil {
		return nil, err
	}
	lo.segments = make([]SegmentInfo, 0, len(segmentInfos))
	for _, info := range segmentInfos {
		lo.segments = append(lo.segments, SegmentInfo{
			Object:    info.Object,
			SizeBytes: info.SizeBytes,
			Etag:      info.Etag,
		})
	}

	return lo, nil
}

func (o *Object) asSLO() (*LargeObject, error) {
	opts := RequestOptions{
		Values: make(url.Values),
	}
	opts.Values.Set("multipart-manifest", "get")
	opts.Values.Set("format", "raw")
	buf, err := o.Download(&opts).AsByteSlice()
	if err != nil {
		return nil, err
	}

	var data []sloSegmentInfo
	err = json.Unmarshal(buf, &data)
	if err != nil {
		return nil, errors.New("invalid SLO manifest: " + err.Error())
	}

	lo := &LargeObject{
		Object:   o,
		Strategy: StaticLargeObject,
	}
	if len(data) == 0 {
		return lo, nil
	}

	//read the segments first, then deduce the SegmentContainer/SegmentPrefix from these
	lo.segments = make([]SegmentInfo, 0, len(data))
	for _, info := range data {
		//option 1: data segment
		if info.DataBase64 != "" {
			data, err := base64.StdEncoding.DecodeString(info.DataBase64)
			if err != nil {
				return nil, errors.New("invalid SLO data segment: " + err.Error())
			}
			lo.segments = append(lo.segments, SegmentInfo{Data: data})
			continue
		}

		//option 2: segment backed by object
		pathElements := strings.SplitN(strings.TrimPrefix(info.Path, "/"), "/", 2)
		if len(pathElements) != 2 {
			return nil, errors.New("invalid SLO segment: malformed path: " + info.Path)
		}
		s := SegmentInfo{
			Object:    o.c.a.Container(pathElements[0]).Object(pathElements[1]),
			SizeBytes: info.SizeBytes,
			Etag:      info.Etag,
		}
		if info.Range != "" {
			var ok bool
			s.RangeOffset, s.RangeLength, ok = parseHTTPRange(info.Range)
			if !ok {
				return nil, errors.New("invalid SLO segment: malformed range: " + info.Range)
			}
		}
		lo.segments = append(lo.segments, s)
	}

	//choose the SegmentContainer by majority vote (in the spirit of "be liberal
	//in what you accept")
	containerNames := make(map[string]uint)
	for _, s := range lo.segments {
		if s.Object == nil { //can happen for data segments
			continue
		}
		containerNames[s.Object.c.Name()]++
	}
	maxName := ""
	maxVotes := uint(0)
	for name, votes := range containerNames {
		if votes > maxVotes {
			maxName = name
			maxVotes = votes
		}
	}
	lo.SegmentContainer = lo.Object.c.a.Container(maxName)

	//choose the SegmentPrefix as the longest common prefix of all segments in
	//the chosen SegmentContainer...
	names := make([]string, 0, len(lo.segments))
	for _, s := range lo.segments {
		if s.Object == nil { //can happen for data segments
			continue
		}
		name := s.Object.c.Name()
		if name == maxName {
			names = append(names, s.Object.Name())
		}
	}
	lo.SegmentPrefix = longestcommon.Prefix(names)

	//..BUT if the prefix is a path with slashes, do not consider the part after
	//the last slash; e.g. if we have segments "foo/bar/0001" and "foo/bar/0002",
	//the longest common prefix is "foo/bar/000", but we actually want "foo/bar/"
	if strings.Contains(lo.SegmentPrefix, "/") {
		lo.SegmentPrefix = path.Dir(lo.SegmentPrefix) + "/"
	}

	return lo, nil
}

func parseHTTPRange(str string) (offsetVal int64, lengthVal uint64, ok bool) {
	fields := strings.SplitN(str, "-", 2)
	if len(fields) != 2 {
		return 0, 0, false
	}

	if fields[0] == "" {
		//case 1: "-"
		if fields[1] == "" {
			return 0, 0, true
		}

		//case 2: "-N"
		numBytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		return -1, numBytes, true
	}

	firstByte, err := strconv.ParseUint(fields[0], 10, 63) //not 64; needs to be unsigned, but also fit into int64
	if err != nil {
		return 0, 0, false
	}
	if fields[1] == "" {
		//case 3: "N-"
		return int64(firstByte), 0, true
	}
	//case 4: "M-N"
	lastByte, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil || lastByte < firstByte {
		return 0, 0, false
	}
	return int64(firstByte), lastByte - firstByte + 1, true
}

//Open returns an io.WriteCloser that can be used to replace or extend the
//contents of this large object.
//
//This call returns ErrNoContainerName if o.SegmentContainer is not set, or
//ErrAccountMismatch if it is not in the same account as the large object.
//For existing objects, SegmentContainer and SegmentPrefix will be filled by
//Object.AsLargeObject(). For new objects, they need to be filled by the
//caller.
//
//WARNING: Every call to Write() on the returned writer will create a new
//segment. To ensure a uniform segment size, wrap the writer returned from this
//call in a bufio.Writer, for example by using the schwift.SetSegmentSize()
//convenience function:
//
//	dlo, err := account.Container("public").Object("archive27.zip").AsLargeObject()
//	dlo.SegmentContainer = account.Container("segments")
//	dlo.SegmentPrefix = "archive27/"
//	w, err := dlo.Open(schwift.OpenTruncate)
//	w, err = schwift.SetSegmentSize(w, 1<<30) //segment size 1<<30 byte = 1 GiB
//	_, err = bw.Write(archiveContents)
//	err = w.Close()
//
func (lo *LargeObject) Open(mode LargeObjectOpenMode) (io.WriteCloser, error) {
	if lo.SegmentContainer == nil {
		return nil, ErrNoContainerName
	}
	if !lo.SegmentContainer.a.isEqualTo(lo.Object.c.a) {
		return nil, ErrAccountMismatch
	}

	if mode&OpenAppend == 0 {
		if mode&OpenKeepSegments == 0 {
			_, _, err := lo.Object.c.a.BulkDelete(lo.segmentObjects(), nil, nil)
			if err != nil {
				return nil, err
			}
		}
		lo.segments = nil
	}

	return largeObjectWriter{lo}, nil
}

//Segments returns a list of all segments for this object, in order.
func (lo *LargeObject) Segments() ([]SegmentInfo, error) {
	//NOTE: This method has an error return value because we might later switch
	//to loading segments lazily inside this method.
	return lo.segments, nil
}

func (lo *LargeObject) segmentObjects() []*Object {
	seen := make(map[string]bool)
	result := make([]*Object, 0, len(lo.segments))
	for _, segment := range lo.segments {
		if segment.Object == nil { //can happen because of data segments
			continue
		}
		fullName := segment.Object.FullName()
		if !seen[fullName] {
			result = append(result, segment.Object)
		}
		seen[fullName] = true
	}

	return result
}

//NextSegmentObject suggests where to upload the next segment.
//
//WARNING: This is a low-level function. Most callers will want to use the
//io.WriteCloser provided by Open(). You will only need to upload segments
//manually when you want to control the segments' metadata.
//
//If the name of the current final segment ends with a counter, that counter is
//incremented, otherwise a counter is appended to its name. When looking for a
//counter in an existing segment name, the regex /[0-9]+$/ is used. For example,
//given:
//
//	segments := lo.Segments()
//	lastSegmentName := segments[len(segments)-1].Name()
//	nextSegmentName := lo.NextSegmentObject().Name()
//
//If lastSegmentName is "segments/archive/segment0001", then nextSegmentName is
//"segments/archive/segment0002". If lastSegmentName is
//"segments/archive/first", then nextSegmentName is
//"segments/archive/first0000000000000001".
//
//However, the last segment's name will only be considered if it lies within
//lo.SegmentContainer below lo.SegmentPrefix. If that is not the case, the name
//of the last segment that does will be used instead.
//
//If there are no segments yet, or if all segments are located outside the
//lo.SegmentContainer and lo.SegmentPrefix, the first segment name is chosen as
//lo.SegmentPrefix + "0000000000000001".
func (lo *LargeObject) NextSegmentObject() *Object {
	//find the name of the last-most segment that is within the designated
	//segment container and prefix
	var prevSegmentName string
	for _, s := range lo.segments {
		o := s.Object
		if o == nil { //can happen for data segments
			continue
		}
		if lo.SegmentContainer.isEqualTo(o.c) && strings.HasPrefix(o.Name(), lo.SegmentPrefix) {
			prevSegmentName = s.Object.Name()
			//keep going, we want to find the last such segment
		}
	}

	//choose the next segment name based on the previous one
	var segmentName string
	if prevSegmentName == "" {
		segmentName = lo.SegmentPrefix + initialIndex
	} else {
		segmentName = nextSegmentName(prevSegmentName)
	}

	return lo.SegmentContainer.Object(segmentName)
}

var splitSegmentIndexRx = regexp.MustCompile(`^(.*?)([0-9]+$)`)
var initialIndex = "0000000000000001"

//Given the object name of a previous large object segment, compute a suitable
//name for the next segment. See doc for LargeObject.NextSegmentObject()
//for how this works.
func nextSegmentName(segmentName string) string {
	match := splitSegmentIndexRx.FindStringSubmatch(segmentName)
	if match == nil {
		return segmentName + initialIndex
	}
	base, idxStr := match[1], match[2]

	idx, err := strconv.ParseUint(idxStr, 10, 64)
	if err != nil || idx == math.MaxUint64 { //overflow
		//start from one again, but separate with a dash to ensure that the new
		//index can be parsed properly in the next call to this function
		return segmentName + "-" + initialIndex
	}

	//print next index with same number of digits as previous index,
	//e.g. "00001" -> "00002" (except if overflow, e.g. "9999" -> "10000")
	formatStr := fmt.Sprintf("%%0%dd", len(idxStr))
	return base + fmt.Sprintf(formatStr, idx+1)
}

//AddSegment appends a segment to this object. The segment must already have
//been uploaded.
//
//WARNING: This is a low-level function. Most callers will want to use the
//io.WriteCloser provided by Open(). You will only need to add segments
//manually when you want to control the segments' metadata, or when using
//advanced features such as range-limited segments or data segments.
//
//This method returns ErrAccountMismatch if the segment is not located in a
//container in the same account.
//
//For dynamic large objects, this method returns ErrContainerMismatch if the
//segment is not located in the correct container below the correct prefix.
//
//This method returns ErrSegmentInvalid if:
//
//- a range is specified in the SegmentInfo, but it is invalid or the
//LargeObject is a dynamic large object (DLOs do not support ranges), or
//
//- the SegmentInfo's Data attribute is set and any other attribute is also
//set (segments cannot be backed by objects and be data segments at the same
//time), or
//
//- the SegmentInfo's Data attribute is set, but the LargeObject is a dynamic
//large objects (DLOs do not support data segments).
func (lo *LargeObject) AddSegment(segment SegmentInfo) error {
	if len(segment.Data) == 0 {
		//validate segments backed by objects
		o := segment.Object
		if o == nil {
			//required attributes
			return ErrSegmentInvalid
		}
		if !o.c.a.isEqualTo(lo.SegmentContainer.a) {
			return ErrAccountMismatch
		}

		switch lo.Strategy {
		case DynamicLargeObject:
			if segment.RangeLength != 0 || segment.RangeOffset != 0 {
				//not supported for DLO
				return ErrSegmentInvalid
			}

			if !o.c.isEqualTo(lo.SegmentContainer) {
				return ErrContainerMismatch
			}
			if !strings.HasPrefix(o.name, lo.SegmentPrefix) {
				return ErrContainerMismatch
			}

		case StaticLargeObject:
			if segment.RangeLength == 0 && segment.RangeOffset < 0 {
				//malformed range
				return ErrSegmentInvalid
			}
		}
	} else {
		//validate plain-data segments
		if lo.Strategy != StaticLargeObject {
			//not supported for DLO
			return ErrSegmentInvalid
		}
		if segment.Object != nil || segment.SizeBytes != 0 || segment.Etag != "" || segment.RangeLength != 0 || segment.RangeOffset != 0 {
			//all other attributes must be unset
			return ErrSegmentInvalid
		}
	}

	lo.segments = append(lo.segments, segment)
	return nil
}

//WriteManifest creates this large object by writing a manifest to its
//location using a PUT request.
//
//For dynamic large objects, this method does not generate a PUT request
//if the object already exists and has the correct manifest (i.e.
//SegmentContainer and SegmentPrefix have not been changed).
func (lo *LargeObject) WriteManifest(opts *RequestOptions) error {
	switch lo.Strategy {
	case StaticLargeObject:
		return lo.writeSLOManifest(opts)
	case DynamicLargeObject:
		return lo.writeDLOManifest(opts)
	default:
		panic("no such strategy")
	}
}

func (lo *LargeObject) writeDLOManifest(opts *RequestOptions) error {
	manifest := lo.SegmentContainer.Name() + "/" + lo.SegmentPrefix

	//check if the manifest is already set correctly
	headers, err := lo.Object.Headers()
	if err != nil && !Is(err, http.StatusNotFound) {
		return err
	}
	if headers.Get("X-Object-Manifest") == manifest {
		return nil
	}

	//write manifest; make sure that this is a DLO
	opts = cloneRequestOptions(opts, nil)
	opts.Headers.Set("X-Object-Manifest", manifest)
	return lo.Object.Upload(nil, opts)
}

func (lo *LargeObject) writeSLOManifest(opts *RequestOptions) error {
	sloSegments := make([]sloSegmentInfo, len(lo.segments))
	for idx, s := range lo.segments {
		if len(s.Data) > 0 {
			sloSegments[idx] = sloSegmentInfo{
				DataBase64: base64.StdEncoding.EncodeToString(s.Data),
			}
		} else {
			si := sloSegmentInfo{
				Path:      "/" + s.Object.FullName(),
				SizeBytes: s.SizeBytes,
				Etag:      s.Etag,
			}

			if s.RangeOffset < 0 {
				si.Range = "-" + strconv.FormatUint(s.RangeLength, 10)
			} else {
				firstByteStr := strconv.FormatUint(uint64(s.RangeOffset), 10)
				lastByteStr := strconv.FormatUint(uint64(s.RangeOffset)+s.RangeLength-1, 10)
				si.Range = firstByteStr + "-" + lastByteStr
			}

			sloSegments[idx] = si
		}
	}

	manifest, err := json.Marshal(sloSegments)
	if err != nil {
		//failing json.Marshal() on such a trivial data structure is alarming
		panic(err.Error())
	}

	opts = cloneRequestOptions(opts, nil)
	opts.Headers.Del("X-Object-Manifest") //ensure sanity :)
	opts.Values.Set("multipart-manifest", "put")
	return lo.Object.Upload(bytes.NewReader(manifest), opts)
}

////////////////////////////////////////////////////////////////////////////////

type largeObjectWriter struct {
	lo *LargeObject
}

//Write implements the io.WriteCloser interface.
func (w largeObjectWriter) Write(buf []byte) (int, error) {
	segment := w.lo.NextSegmentObject()
	//TODO: split write into multiple segments if len(buf) > max object size
	err := segment.Upload(bytes.NewReader(buf), nil)
	if err != nil {
		return 0, err
	}

	sum := md5.Sum(buf)
	return len(buf), w.lo.AddSegment(SegmentInfo{
		Object:    segment,
		SizeBytes: uint64(len(buf)),
		Etag:      hex.EncodeToString(sum[:]),
	})
}

//Close implements the io.WriteCloser interface.
func (w largeObjectWriter) Close() error {
	return w.lo.WriteManifest(nil)
}

////////////////////////////////////////////////////////////////////////////////

type largeObjectBufferedWriter struct {
	bw *bufio.Writer
	w  io.WriteCloser
}

//SetSegmentSize creates a bufio.Writer around an io.WriteCloser and returns
//an interface to it that works like the original io.WriteCloser.
//
//This is intended to be used when writing segments into a large object.
//The writer returned by LargeObject.Open() does not ensure a uniform segment
//size by default, so one would have to wrap it in a bufio.Writer like so:
//
//	dlo, err := account.Container("public").Object("archive27.zip").AsLargeObject()
//	dlo.SegmentContainer = account.Container("segments")
//	dlo.SegmentPrefix = "archive27/"
//
//	w, err := largeObject.Open(schwift.OpenTruncate)
//	bw, err := bufio.NewWriterSize(w, 1<<30) //segment size 1<<30 byte = 1 GiB
//	_, err = bw.Write(archiveContents)
//	err = bw.Flush()
//	err = w.Close()
//
//This function reduces the boilerplate to:
//
//	w, err := largeObject.Open(schwift.OpenTruncate)
//	w, err = schwift.SetSegmentSize(w, 1<<30) //segment size 1<<30 byte = 1 GiB
//	_, err = w.Write(archiveContents)
//	err = w.Close()
//
//Another advantage of this function is that the returned writer implements
//io.WriteCloser, which bufio.Writer does not. So you can pass it into
//consuming functions that use io.WriteCloser to close the object once they're
//done writing to it, and it will be ensured that the buffer is flushed before
//closing the underlying writer.
func SetSegmentSize(w io.WriteCloser, segmentSizeBytes int) io.WriteCloser {
	switch w := w.(type) {
	case *largeObjectBufferedWriter:
		//never chain multiple largeObjectBufferedWriter together
		w.bw.Flush() //ensure that previous calls to `w.Write()` are durable
		return SetSegmentSize(w.w, segmentSizeBytes)
	default:
		return &largeObjectBufferedWriter{
			bw: bufio.NewWriterSize(w, segmentSizeBytes),
			w:  w,
		}
	}
}

//Write implements the io.WriteCloser interface.
func (bw *largeObjectBufferedWriter) Write(buf []byte) (int, error) {
	return bw.bw.Write(buf)
}

//Close implements the io.WriteCloser interface.
func (bw *largeObjectBufferedWriter) Close() error {
	err := bw.bw.Flush()
	if err != nil {
		return err
	}
	return bw.w.Close()
}
