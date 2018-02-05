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

package headers

import (
	"math"
	"strconv"
	"time"
)

//UnixTimeReadonly is a helper type that provides type-safe access to a Swift
//header whose value is a UNIX timestamp. It cannot be directly constructed,
//but some subtypes of schwift.Headers have fields of this type. For example:
//
//    var hdr AccountHeaders
//    //hdr.Timestamp is a headers.UnixTimeReadonly instance
//    hdr.Timestamp.Get()    //returns a time.Time
//    hdr.Get("X-Timestamp") //returns a string containing a UNIX timestamp
//                           //refering to the same point in time
type UnixTimeReadonly struct {
	Base
}

//Exists checks whether there is a value for this header.
func (f UnixTimeReadonly) Exists() bool {
	return f.H.Get(f.K) != ""
}

//Get returns the value for this header, or the zero value if there is no value
//(or if it is not a valid timestamp).
func (f UnixTimeReadonly) Get() time.Time {
	v, err := strconv.ParseFloat(f.H.Get(f.K), 64)
	if err != nil {
		return time.Time{}
	}
	seconds := math.Floor(v)
	return time.Unix(
		int64(seconds),
		int64(1e9*(v-seconds)),
	)
}

//Validate is only used internally, but needs to be exported to cross package
//boundaries.
func (f UnixTimeReadonly) Validate() error {
	val := f.H.Get(f.K)
	if val == "" {
		return nil
	}
	_, err := strconv.ParseFloat(val, 64)
	if err == nil {
		return nil
	}
	return MalformedHeaderError{f.K, err}
}
