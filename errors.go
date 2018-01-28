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
	return fmt.Sprintf("expected %s response, got %d instead: %s",
		strings.Join(codeStrs, "/"),
		e.ActualResponse.StatusCode,
		string(e.ResponseBody),
	)
}

//Is checks if the given error is an UnexpectedStatusCodeError for that status
//code. For example:
//
//	metadata, err := container.Metadata()
//	if schwift.Is(err, http.StatusNotFound) {
//		// ... create container ...
//	} else if err != nil {
//		// ... report error ...
//	} else {
//		// ... use metadata ...
//	}
func Is(err error, code int) bool {
	if e, ok := err.(UnexpectedStatusCodeError); ok {
		return e.ActualResponse.StatusCode == code
	}
	return false
}
