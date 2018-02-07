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
	"net/http"
	"net/textproto"
)

func headersToHTTP(h map[string]string) http.Header {
	if h == nil {
		return nil
	}
	dest := make(http.Header, len(h))
	for k, v := range h {
		dest.Set(k, v)
	}
	return dest
}

func headersFromHTTP(src http.Header) map[string]string {
	if src == nil {
		return nil
	}
	h := make(map[string]string, len(src))
	for k, v := range src {
		if len(v) > 0 {
			h[textproto.CanonicalMIMEHeaderKey(k)] = v[0]
		}
	}
	return h
}
