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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var (
	//ErrChecksumMismatch is returned by Object.Upload() when the Etag in the
	//server response does not match the uploaded data.
	ErrChecksumMismatch = errors.New("Etag on uploaded object does not match MD5 checksum of uploaded data")
	//ErrNoContainerName is returned by Request.Do() if ObjectName is given, but
	//ContainerName is empty.
	ErrNoContainerName = errors.New("missing container name")
	//ErrMalformedContainerName is returned by Request.Do() if ContainerName
	//contains slashes.
	ErrMalformedContainerName = errors.New("container name may not contain slashes")
)

//UnexpectedStatusCodeError is generated when a request to Swift does not yield
//a response with the expected successful status code.
type UnexpectedStatusCodeError struct {
	ExpectedStatusCodes []int
	ActualResponse      *http.Response
	ResponseBody        []byte
}

//Error implements the builtin/error interface.
func (e UnexpectedStatusCodeError) Error() string {
	codeStrs := make([]string, len(e.ExpectedStatusCodes))
	for idx, code := range e.ExpectedStatusCodes {
		codeStrs[idx] = strconv.Itoa(code)
	}
	msg := fmt.Sprintf("expected %s response, got %d instead",
		strings.Join(codeStrs, "/"),
		e.ActualResponse.StatusCode,
	)
	if len(e.ResponseBody) > 0 {
		msg += ": " + string(e.ResponseBody)
	}
	return msg
}

//Is checks if the given error is an UnexpectedStatusCodeError for that status
//code. For example:
//
//	err := container.Delete(nil, nil)
//	if err != nil {
//	    if schwift.Is(err, http.StatusNotFound) {
//	        //container does not exist -> just what we wanted
//	        return nil
//	    } else {
//	        //report unexpected error
//	        return err
//	    }
//	}
func Is(err error, code int) bool {
	if e, ok := err.(UnexpectedStatusCodeError); ok {
		return e.ActualResponse.StatusCode == code
	}
	return false
}

//MalformedHeaderError is generated when a response from Swift contains a
//malformed header.
type MalformedHeaderError struct {
	Key        string
	ParseError error
}

//Error implements the builtin/error interface.
func (e MalformedHeaderError) Error() string {
	return "Bad header " + e.Key + ": " + e.ParseError.Error()
}
