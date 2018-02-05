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

//Metadata is a helper type that provides safe access to the metadata headers
//in a schwift.Headers instance. It cannot be directly constructed, but each
//subtype of schwift.Headers has a field "Metadata" of this type. For example:
//
//    var hdr ObjectHeaders
//    //the following two statements are equivalent
//    hdr.Set("X-Object-Meta-Access", "strictly confidential")
//    hdr.Metadata.Set("Access", "strictly confidential")
//    //because hdr.Metadata is a headers.Metadata instance
type Metadata struct {
	Base
}

//Clear works like Headers.Clear(), but prepends the metadata prefix to the key.
func (m Metadata) Clear(key string) {
	m.H.Clear(m.K + key)
}

//Del works like Headers.Del(), but prepends the metadata prefix to the key.
func (m Metadata) Del(key string) {
	m.H.Del(m.K + key)
}

//Get works like Headers.Get(), but prepends the metadata prefix to the key.
func (m Metadata) Get(key string) string {
	return m.H.Get(m.K + key)
}

//Set works like Headers.Set(), but prepends the metadata prefix to the key.
func (m Metadata) Set(key, value string) {
	m.H.Set(m.K+key, value)
}
