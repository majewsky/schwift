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
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type tempurlBogusBackend struct {
	mockInfoText string
}

func (tempurlBogusBackend) EndpointURL() string {
	return "https://example.com/v1/AUTH_example/"
}
func (tempurlBogusBackend) Clone(newEndpointURL string) Backend {
	panic("unimplemented")
}
func (tBB tempurlBogusBackend) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/info" {
		reader := strings.NewReader(tBB.mockInfoText)
		return &http.Response{Body: io.NopCloser(reader)}, nil
	}
	panic("unimplemented")
}

func expectString(t *testing.T, expected, actual string) {
	if actual != expected {
		t.Error("temp URL generation failed")
		t.Logf("expected: %s\n", expected)
		t.Logf("actual: %s\n", actual)
	}
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestObjectTempURLSha1Only(t *testing.T) {
	// setup a bogus backend, account, container and object with exact names to
	// reproducibly generate a temp URL
	account, err := InitializeAccount(tempurlBogusBackend{
		mockInfoText: `{ "tempurl": { "allowed_digests": [ "sha1" ]}}`,
	})
	must(t, err)

	actualURL, err := account.Container("foo").Object("bar").TempURL(context.TODO(), "supersecretkey", "GET", time.Unix(1e9, 0))
	must(t, err)

	expectedURL := "https://example.com/v1/AUTH_example/foo/bar?temp_url_sig=ed44d92005345aee463c884d76d4850ef6d2778d&temp_url_expires=1000000000"
	expectString(t, expectedURL, actualURL)
}

func TestObjectTempURL(t *testing.T) {
	// setup a bogus backend, account, container and object with exact names to
	// reproducibly generate a temp URL
	account, err := InitializeAccount(tempurlBogusBackend{
		mockInfoText: `{ "tempurl": { "allowed_digests": [ "sha1", "sha256", "sha512"]}}`,
	})
	must(t, err)

	actualURL, err := account.Container("foo").Object("bar").TempURL(context.TODO(), "supersecretkey", "GET", time.Unix(1e9, 0))
	must(t, err)

	expectedURL := "https://example.com/v1/AUTH_example/foo/bar?temp_url_sig=5fc94a988b502d83e88863774812636ef0133b8aae04b20366fd906bff41189f&temp_url_expires=1000000000"
	expectString(t, expectedURL, actualURL)
}
