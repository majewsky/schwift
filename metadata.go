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

import "net/textproto"

//Metadata works like http.Header, but does not allow multiple values per key.
type Metadata map[string]string

//NewMetadata constructs a Metadata instance from a list of key-value pairs
//with compact syntax. It is recommended over a map literal since it correctly
//formats keys with textproto.CanonicalMIMEHeaderKey(). For example:
//
//    m = NewMetadata(
//        "color", "blue",
//        "size", "large",
//    )
//
//    //...is equivalent to...
//
//    m = make(Metadata)
//    m.Set("color", "blue")
//    m.Set("size", "large")
//
//NewMetadata panics if it is called with an odd number of arguments.
func NewMetadata(args ...string) Metadata {
	if len(args)%2 == 1 {
		panic("NewMetadata called with an odd number of arguments")
	}
	m := make(Metadata)
	for idx := 0; idx < len(args); idx += 2 {
		m.Set(args[idx], args[idx+1])
	}
	return m
}

//Clear sets the value to this key to the empty string, such that a Post() with
//this Metadata will remove the existing value from this metadata key on the server.
func (m Metadata) Clear(key string) {
	m.Set(key, "")
}

//Del works just like http.Header.Del().
//
//Del deletes a key from the Metadata instance. When the Metadata instance
//is then sent to the server with Post(), Del() has different effects depending
//on context because of Swift's inconsistent API:
//
//For account or container metadata, a key which has been deleted with Del() will
//remain unchanged on the server. To remove the key on the server, use Clear()
//instead.
//
//For object metadata, deleting a key will cause that key to be deleted on the
//server. Del() is identical to Clear() in this case.
func (m Metadata) Del(key string) {
	k := textproto.CanonicalMIMEHeaderKey(key)
	delete(m, k)
}

//Get works just like http.Header.Get().
func (m Metadata) Get(key string) string {
	if m == nil {
		return ""
	}
	k := textproto.CanonicalMIMEHeaderKey(key)
	return m[k]
}

//Set works just like http.Header.Set().
func (m Metadata) Set(key, value string) {
	k := textproto.CanonicalMIMEHeaderKey(key)
	m[k] = value
}
