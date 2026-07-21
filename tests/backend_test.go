// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"net/http"

	"go.xyrillian.de/schwift/v2"
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
