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
	"fmt"
	"net/http"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
)

//AccountHeaders contains the headers for an account. The Raw attribute
//contains the original set of headers returned from a HEAD or GET request on
//the account. The other attributes contain the parsed values of common
//headers, as noted in the tags next to each field. Well-known metadata headers
//can be accessed in a type-safe way using the methods on this type.
type AccountHeaders struct {
	BytesUsed      uint64   `schwift:"ro,X-Account-Bytes-Used"`
	ContainerCount uint64   `schwift:"ro,X-Account-Container-Count"`
	ObjectCount    uint64   `schwift:"ro,X-Account-Object-Count"`
	Metadata       Metadata `schwift:"rw,X-Account-Meta-"`
	Raw            http.Header
}

//QuotaBytes returns a handle to read or write the X-Account-Meta-Quota-Bytes field.
func (a AccountHeaders) QuotaBytes() UnsignedIntField {
	return UnsignedIntField{
		a.Metadata,
		"X-Account-Meta-", "Quota-Bytes",
	}
}

//TempURLKey returns a handle to read or write the X-Account-Meta-Temp-URL-Key field.
func (a AccountHeaders) TempURLKey() StringField {
	return StringField{
		a.Metadata,
		"X-Account-Meta-", "Temp-URL-Key",
	}
}

//TempURLKey2 returns a handle to read or write the X-Account-Meta-Temp-URL-Key-2 field.
func (a AccountHeaders) TempURLKey2() StringField {
	return StringField{
		a.Metadata,
		"X-Account-Meta-", "Temp-URL-Key-2",
	}
}

//ContainerHeaders contains the headers for an account. The Raw attribute
//contains the original set of headers returned from a HEAD or GET request on
//the account. The other attributes contain the parsed values of common
//headers, as noted in the tags next to each field. Well-known metadata headers
//can be accessed in a type-safe way using the methods on this type.
type ContainerHeaders struct {
	Metadata Metadata `schwift:"rw,X-Container-Meta-"`
	Raw      http.Header
	//TODO map well-known headers
}

////////////////////////////////////////////////////////////////////////////////
// field types

//StringField is a helper type used in the interface of AccountHeaders,
//ContainerHeaders and ObjectHeaders. For example:
//
//    var headers AccountHeaders
//    ...
//    value := headers.TempURLKey().Get()
//    headers.TempURLKey().Set(value + " changed")
//    headers.TempURLKey().Clear()
type StringField struct {
	metadata Metadata
	prefix   string
	key      string
}

//Get returns the value for this key, or the empty string if the key does not exist.
func (f StringField) Get() string {
	return f.metadata.Get(f.key)
}

//Set writes a new value for this key into the original AccountHeaders,
//ContainerHeaders or ObjectHeaders instance.
func (f StringField) Set(value string) {
	f.metadata.Set(f.key, value)
}

//Clear removes this key from the original AccountHeaders, ContainerHeaders or
//ObjectHeaders instance.
func (f StringField) Clear() {
	f.metadata.Clear(f.key)
}

//UnsignedIntField is a helper type used in the interface of AccountHeaders,
//ContainerHeaders and ObjectHeaders. For example:
//
//    var headers AccountHeaders
//    ...
//    if headers.QuotaBytes.Exists() {
//        value, err := headers.QuotaBytes().Get()
//        headers.QuotaBytes().Set(value * 2)
//    }
//    ....
//    headers.QuotaBytes().Clear()
type UnsignedIntField struct {
	metadata Metadata
	prefix   string
	key      string
}

//Exists returns whether there is a value for this key.
func (f UnsignedIntField) Exists() bool {
	return f.metadata.Get(f.key) != ""
}

//Get returns the value for this key, or 0 if the key does not exist.
func (f UnsignedIntField) Get() (uint64, error) {
	str := f.metadata.Get(f.key)
	if str == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		err = MalformedHeaderError{Key: f.prefix + f.key, ParseError: err}
	}
	return value, err
}

//Set writes a new value for this key into the original AccountHeaders,
//ContainerHeaders or ObjectHeaders instance.
func (f UnsignedIntField) Set(value uint64) {
	f.metadata.Set(f.key, strconv.FormatUint(value, 10))
}

//Clear removes this key from the original AccountHeaders, ContainerHeaders or
//ObjectHeaders instance.
func (f UnsignedIntField) Clear() {
	f.metadata.Clear(f.key)
}

////////////////////////////////////////////////////////////////////////////////
// generic parsing functions

func parseHeaders(hdr http.Header, target interface{}) error {
	return foreachField(target, func(fieldPtr interface{}, info fieldInfo) error {
		//populate the .Raw field that all input types share
		if info.FieldName == "Raw" {
			*(fieldPtr.(*http.Header)) = hdr
			return nil
		}

		//skip over fields without schwift field tag
		if info.HeaderName == "" {
			return nil
		}

		//decode header value into field depending on type
		switch fieldPtr := fieldPtr.(type) {
		case *string:
			*fieldPtr = hdr.Get(info.HeaderName)
		case *uint64:
			value, err := strconv.ParseUint(hdr.Get(info.HeaderName), 10, 64)
			if err != nil {
				return MalformedHeaderError{info.HeaderName, err}
			}
			*fieldPtr = value
		case *Metadata:
			//collect all headers with a prefix equal to `headerName`
			values := make(Metadata)
			for key, value := range hdr {
				key = textproto.CanonicalMIMEHeaderKey(key)
				if strings.HasPrefix(key, info.HeaderName) {
					key = strings.TrimPrefix(key, info.HeaderName)
					values[key] = value[0]
				}
			}
			*fieldPtr = values
		default:
			panic(fmt.Sprintf("parseHeaders: cannot handle field type %T", fieldPtr))
		}

		return nil
	})
}

func compileHeaders(headers interface{}, opts *RequestOptions) RequestOptions {
	hdr := make(http.Header)

	foreachField(headers, func(fieldPtr interface{}, info fieldInfo) error {
		//skip over fields without schwift field tag, and readonly fields
		if info.HeaderName == "" || info.Access != "rw" {
			return nil
		}

		//decode header value into field depending on type
		switch fieldPtr := fieldPtr.(type) {
		case *string:
			hdr.Set(info.HeaderName, *fieldPtr)
		case *uint64:
			hdr.Set(info.HeaderName, strconv.FormatUint(*fieldPtr, 10))
		case *Metadata:
			for key, value := range *fieldPtr {
				//empty string means that this key shall be removed
				if value == "" {
					//for object metadata, a key is removed by just omitting it...
					if info.HeaderName != "X-Object-Meta-" {
						//...for container and account metadata, a key is removed by
						//setting its value to the empty string
						hdr.Set(info.HeaderName+key, "")
					}
				} else {
					//NOTE: The spec says that `value` needs to be percent-encoded, but
					//neither python-swiftclient nor ncw/swift do so. If in doubt, we
					//follow the de-facto standards rather than the spec.
					hdr.Set(info.HeaderName+key, value)
				}
			}
		default:
			panic(fmt.Sprintf("compileHeaders: cannot handle field type %T", fieldPtr))
		}
		return nil
	})

	//contents of `opts` overrides contents of `headers`
	result := RequestOptions{Headers: hdr}
	if opts != nil {
		result.Values = opts.Values
		for k, v := range opts.Headers {
			result.Headers[k] = v
		}
	}
	return result
}

type fieldInfo struct {
	FieldName  string
	Access     string
	HeaderName string
}

func foreachField(value interface{}, callback func(fieldPtr interface{}, info fieldInfo) error) error {
	rv := reflect.ValueOf(value)
	//unpack pointer type if necessary
	if rv.Type().Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	//iterate over the struct fields
	for idx := 0; idx < rv.NumField(); idx++ {
		fieldType := rv.Type().Field(idx)
		fieldPtr := rv.Field(idx).Addr().Interface()

		//decode schwift:"<access>,<header-name>" tag
		tagValues := strings.SplitN(fieldType.Tag.Get("schwift"), ",", 2)
		fieldInfo := fieldInfo{
			FieldName: fieldType.Name,
		}
		if len(tagValues) == 2 {
			fieldInfo.Access = tagValues[0]
			fieldInfo.HeaderName = tagValues[1]
		}

		err := callback(fieldPtr, fieldInfo)
		if err != nil {
			return err
		}
	}

	return nil
}
