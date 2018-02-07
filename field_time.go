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
	"strconv"
	"time"
)

//FieldUnixTimeReadonly is a helper type that provides type-safe access to a
//Swift header whose value is a UNIX timestamp. It cannot be directly
//constructed, but methods on the Headers types return this type. For example:
//
//    //suppose you have:
//    hdr, err := obj.Headers()
//
//    //you could do all this:
//    sec, err := strconv.ParseFloat(hdr.Get("X-Timestamp"), 64)
//    time := time.Unix(int64(sec), int64(1e9 * (sec - math.Floor(sec))))
//
//    //or you can just:
//    time := hdr.Timestamp().Get()
//
//Don't worry about the missing `err` in the last line. When the X-Timestamp
//header fails to parse, Object.Headers() already returns the corresponding
//MalformedHeaderError.
type FieldUnixTimeReadonly struct {
	h headerInterface
	k string
}

//Exists checks whether there is a value for this header.
func (f FieldUnixTimeReadonly) Exists() bool {
	return f.h.Get(f.k) != ""
}

//Get returns the value for this header, or the zero value if there is no value
//(or if it is not a valid timestamp).
func (f FieldUnixTimeReadonly) Get() time.Time {
	v, err := strconv.ParseFloat(f.h.Get(f.k), 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(0, int64(1e9*v))
}

func (f FieldUnixTimeReadonly) validate() error {
	val := f.h.Get(f.k)
	if val == "" {
		return nil
	}
	_, err := strconv.ParseFloat(val, 64)
	if err == nil {
		return nil
	}
	return MalformedHeaderError{f.k, err}
}
