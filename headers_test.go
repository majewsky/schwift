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
	"testing"
)

func TestParseAccountHeadersSuccess(t *testing.T) {
	headers := NewAccountHeaders()
	headers.FromHTTP(http.Header{
		"X-Account-Bytes-Used":       {"1234"},
		"X-Account-Object-Count":     {"42"},
		"X-Account-Container-Count":  {"23"},
		"X-Account-Meta-Quota-Bytes": {"1048576"},
		"X-Account-Meta-foo":         {"bar"},
	})

	expectError(t, headers.Validate(), "")
	expectUint64(t, headers.BytesUsed.Get(), 1234)
	expectUint64(t, headers.ContainerCount.Get(), 23)
	expectUint64(t, headers.ObjectCount.Get(), 42)
	expectUint64(t, headers.QuotaBytes.Get(), 1048576)

	expectString(t, headers.Metadata.Get("foo"), "bar")
	expectString(t, headers.Metadata.Get("Foo"), "bar")
	expectString(t, headers.Metadata.Get("FOO"), "bar")
}

//TODO TestParseAccountHeadersError

func expectBool(t *testing.T, actual bool, expected bool) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected value %#v, got %#v instead\n", expected, actual)
	}
}

func expectUint64(t *testing.T, actual uint64, expected uint64) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected value %d, got %d instead\n", expected, actual)
	}
}

func expectString(t *testing.T, actual string, expected string) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected value %q, got %q instead\n", expected, actual)
	}
}

func expectError(t *testing.T, actual error, expected string) (ok bool) {
	t.Helper()
	if actual == nil {
		if expected != "" {
			t.Errorf("expected error %q, got no error\n", expected)
			return false
		}
	} else {
		if expected == "" {
			t.Errorf("expected no error, got %q\n", actual.Error())
			return false
		} else if expected != actual.Error() {
			t.Errorf("expected error %q, got %q instead\n", expected, actual.Error())
			return false
		}
	}

	return true
}
