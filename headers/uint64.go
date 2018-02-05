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
	"strconv"
)

//Uint64 is a helper type that provides type-safe access to a Swift header
//whose value is an unsigned integer. It cannot be directly constructed, but
//some subtypes of schwift.Headers have fields of this type. For example:
//
//    var hdr AccountHeaders
//    //the following two statements are equivalent:
//    hdr.Set("X-Account-Meta-Quota-Bytes", "1048576")
//    hdr.QuotaBytes.Set(1 << 20)
//    //because hdr.QuotaBytes is a headers.Uint64 instance
type Uint64 struct {
	Base
}

//Exists checks whether there is a value for this header.
func (f Uint64) Exists() bool {
	return f.H.Get(f.K) != ""
}

//Get returns the value for this header, or 0 if there is no value (or if it is
//not a valid uint64).
func (f Uint64) Get() uint64 {
	v, err := strconv.ParseUint(f.H.Get(f.K), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

//Set writes a new value for this header into the corresponding schwift.Headers
//instance.
func (f Uint64) Set(value uint64) {
	f.H.Set(f.K, strconv.FormatUint(value, 10))
}

//Del removes this key from the original schwift.Headers instance, so that the
//key will remain unchanged on the server during Update().
func (f Uint64) Del() {
	f.H.Del(f.K)
}

//Clear sets this key to an empty string in the original schwift.Headers
//instance, so that the key will be removed on the server during Update().
func (f Uint64) Clear() {
	f.H.Clear(f.K)
}

//Validate is only used internally, but needs to be exported to cross package
//boundaries.
func (f Uint64) Validate() error {
	val := f.H.Get(f.K)
	if val == "" {
		return nil
	}
	_, err := strconv.ParseUint(val, 10, 64)
	if err == nil {
		return nil
	}
	return MalformedHeaderError{f.K, err}
}

////////////////////////////////////////////////////////////////////////////////

//Uint64Readonly is a readonly variant of Uint64. It is used for fields that
//cannot be set by the client.
type Uint64Readonly struct {
	Base
}

//Exists checks whether there is a value for this header.
func (f Uint64Readonly) Exists() bool {
	return f.H.Get(f.K) != ""
}

//Get returns the value for this header, or 0 if there is no value (or if it is
//not a valid uint64).
func (f Uint64Readonly) Get() uint64 {
	v, err := strconv.ParseUint(f.H.Get(f.K), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

//Validate is only used internally, but needs to be exported to cross package
//boundaries.
func (f Uint64Readonly) Validate() error {
	val := f.H.Get(f.K)
	if val == "" {
		return nil
	}
	_, err := strconv.ParseUint(val, 10, 64)
	if err == nil {
		return nil
	}
	return MalformedHeaderError{f.K, err}
}
