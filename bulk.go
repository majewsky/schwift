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
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

//BulkUploadFormat enumerates possible archive formats for Container.BulkUpload().
type BulkUploadFormat string

const (
	//BulkUploadTar is a plain tar archive.
	BulkUploadTar BulkUploadFormat = "tar"
	//BulkUploadTarGzip is a GZip-compressed tar archive.
	BulkUploadTarGzip BulkUploadFormat = "tar.gz"
	//BulkUploadTarBzip2 is a BZip2-compressed tar archive.
	BulkUploadTarBzip2 BulkUploadFormat = "tar.bz2"
)

//BulkUpload extracts an archive (which may contain multiple files) into a
//Swift account. The path of each file in the archive is appended to the
//uploadPath to form the FullName() of the resulting Object.
//
//For example, when uploading an archive that contains the file "a/b/c":
//
//	//This uploads the file into the container "a" as object "b/c".
//	account.BulkUpload("", format, contents, nil, nil)
//	//This uploads the file into the container "foo" as object "a/b/c".
//	account.BulkUpload("foo", format, contents, nil, nil)
//	//This uploads the file into the container "foo" as object "bar/baz/a/b/c".
//	account.BulkUpload("foo/bar/baz", format, contents, nil, nil)
//
//The first return value indicates the number of files that have been created
//on the server side. This may be lower than the number of files in the archive
//if some files could not be saved individually (e.g. because a quota was
//exceeded in the middle of the archive extraction).
//
//If not nil, the error return value is *usually* an instance of
//BulkError.
//
//This operation returns (0, ErrNotSupported) if the server does not support
//bulk-uploading.
func (a *Account) BulkUpload(uploadPath string, format BulkUploadFormat, contents io.Reader, headers AccountHeaders, opts *RequestOptions) (int, error) {
	caps, err := a.Capabilities()
	if err != nil {
		return 0, err
	}
	if caps.BulkUpload == nil {
		return 0, ErrNotSupported
	}

	req := Request{
		Method:            "PUT",
		Body:              contents,
		Headers:           headersToHTTP(headers),
		Options:           cloneRequestOptions(opts),
		ExpectStatusCodes: []int{200},
	}
	req.Headers.Set("Accept", "application/json")
	req.Options.Values.Set("extract-archive", string(format))

	fields := strings.SplitN(strings.Trim(uploadPath, "/"), "/", 2)
	req.ContainerName = fields[0]
	if len(fields) == 2 {
		req.ObjectName = fields[1]
	}

	resp, err := req.Do(a.backend)
	if err != nil {
		return 0, err
	}

	var result struct {
		//ResponseStatus indicates the overall result as a HTTP status string, e.g.
		//"201 Created" or "500 Internal Error".
		ResponseStatus string `json:"Response Status"`
		//ResponseBody contains an overall error message for errors that are not
		//related to a single file in the archive (e.g. "invalid tar file" or "Max
		//delete failures exceeded").
		ResponseBody string `json:"Response Body"`
		//Errors contains error messages for individual files. Each entry is a
		//[]string with 2 elements, the object's fullName and the HTTP status for
		//this file's upload (e.g. "412 Precondition Failed").
		Errors [][]string `json:"Errors"`
		//NumberFilesCreated is self-explanatory.
		NumberFilesCreated int `json:"Number Files Created"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	closeErr := resp.Body.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}

	//parse `result` into type BulkError
	bulkErr := BulkError{
		OverallError: result.ResponseBody,
	}
	bulkErr.StatusCode, err = parseResponseStatus(result.ResponseStatus)
	if err != nil {
		return 0, err
	}
	for _, suberr := range result.Errors {
		if len(suberr) != 2 {
			continue //wtf
		}
		nameFields := strings.SplitN(suberr[0], "/", 2)
		for len(nameFields) < 2 {
			nameFields = append(nameFields, "")
		}
		statusCode, err := parseResponseStatus(suberr[1])
		if err != nil {
			return 0, err
		}
		bulkErr.ObjectErrors = append(bulkErr.ObjectErrors, BulkObjectError{
			ContainerName: nameFields[0],
			ObjectName:    nameFields[1],
			StatusCode:    statusCode,
		})
	}

	//is BulkError really an error?
	if len(bulkErr.ObjectErrors) == 0 && bulkErr.OverallError == "" && bulkErr.StatusCode >= 200 && bulkErr.StatusCode < 300 {
		return result.NumberFilesCreated, nil
	}
	return result.NumberFilesCreated, bulkErr
}

func parseResponseStatus(status string) (int, error) {
	//`status` looks like "201 Created"
	fields := strings.SplitN(status, " ", 2)
	return strconv.Atoi(fields[0])
}
