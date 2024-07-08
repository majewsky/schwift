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

package tests

import (
	"net/http"

	"github.com/majewsky/schwift/v2"
)

type RequestCountingBackend struct {
	Inner schwift.Backend
	Count int
}

func (b *RequestCountingBackend) EndpointURL() string {
	return b.Inner.EndpointURL()
}

func (b *RequestCountingBackend) Clone(newEndpointURL string) schwift.Backend {
	return &RequestCountingBackend{Inner: b.Inner.Clone(newEndpointURL)}
}

func (b *RequestCountingBackend) Do(req *http.Request) (*http.Response, error) {
	b.Count++
	return b.Inner.Do(req)
}
