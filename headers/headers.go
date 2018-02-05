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

//Package headers contains helper types for the type-safe representation of
//headers on Swift accounts/containers/objects.
package headers

import (
	"net/http"
	"net/textproto"
)

//Headers works like http.Header, but does not allow multiple values per key.
//
//If you write the map directly, without using the provided methods, you must
//normalize all keys with textproto.CanonicalMIMEHeaderKey(). Otherwise, the
//results are undefined.
type Headers map[string]string

//Clear sets the value for the specified header to the empty string. When the
//Headers instance is then sent to the server with Update(), the server will
//delete the value for that header; cf. Del().
func (h Headers) Clear(key string) {
	h.Set(key, "")
}

//Del deletes a key from the Headers instance. When the Headers instance
//is then sent to the server with Update(), Del() has different effects
//depending on context because of Swift's inconsistent API:
//
//For most writable attributes, a key which has been deleted with Del() will
//remain unchanged on the server. To remove the key on the server, use Clear()
//instead.
//
//For object metadata (but not other object attributes), deleting a key will
//cause that key to be deleted on the server. Del() is identical to Clear() in
//this case.
func (h Headers) Del(key string) {
	k := textproto.CanonicalMIMEHeaderKey(key)
	delete(h, k)
}

//Get returns the value for the specified header.
func (h Headers) Get(key string) string {
	if h == nil {
		return ""
	}
	k := textproto.CanonicalMIMEHeaderKey(key)
	return h[k]
}

//Set sets a new value for the specified header, possibly overwriting a
//previous value.
func (h Headers) Set(key, value string) {
	k := textproto.CanonicalMIMEHeaderKey(key)
	h[k] = value
}

//ToHTTP converts this map into a http.Header.
func (h Headers) ToHTTP() http.Header {
	dest := make(http.Header, len(h))
	for k, v := range h {
		dest.Set(k, v)
	}
	return dest
}

//FromHTTP populates this map with the headers in the given http.Header. When a
//header has multiple values, every value but the first one will be discarded.
func (h Headers) FromHTTP(src http.Header) {
	for k, v := range src {
		if len(v) > 0 {
			h.Set(k, v[0])
		}
	}
}
