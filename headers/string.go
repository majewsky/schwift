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

//String is a helper type that provides type-safe access to a Swift header key
//whose value is a string. It cannot be directly constructed, but some subtypes
//of schwift.Headers have fields of this type. For example:
//
//    var hdr AccountHeaders
//    //the following two statements are equivalent:
//    hdr.Set("X-Container-Read", ".r:*,.rlistings")
//    hdr.ReadACL.Set(".r:*,.rlistings")
//    //because hdr.ReadACL is a headers.String instance
type String struct {
	Base
}

//Exists checks whether there is a value for this header.
func (f String) Exists() bool {
	return f.H.Get(f.K) != ""
}

//Get returns the value for this header, or the empty string if there is no value.
func (f String) Get() string {
	return f.H.Get(f.K)
}

//Set writes a new value for this header into the corresponding schwift.Headers
//instance.
func (f String) Set(value string) {
	f.H.Set(f.K, value)
}

//Del removes this key from the original schwift.Headers instance, so that the
//key will remain unchanged on the server during Update().
func (f String) Del() {
	f.H.Del(f.K)
}

//Clear sets this key to an empty string in the original schwift.Headers
//instance, so that the key will be removed on the server during Update().
func (f String) Clear() {
	f.H.Clear(f.K)
}
