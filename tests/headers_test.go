// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"testing"

	"go.xyrillian.de/schwift/v2"
)

func TestParseAccountHeadersSuccess(t *testing.T) {
	headers := schwift.AccountHeaders{
		Headers: schwift.Headers{
			"X-Account-Bytes-Used":       "1234",
			"X-Account-Object-Count":     "42",
			"X-Account-Container-Count":  "23",
			"X-Account-Meta-Quota-Bytes": "1048576",
			"X-Account-Meta-Foo":         "bar",
		},
	}

	expectSuccess(t, headers.Validate())
	expectUint64(t, headers.BytesUsed().Get(), 1234)
	expectUint64(t, headers.ContainerCount().Get(), 23)
	expectUint64(t, headers.ObjectCount().Get(), 42)
	expectUint64(t, headers.BytesUsedQuota().Get(), 1048576)

	expectString(t, headers.Metadata().Get("foo"), "bar")
	expectString(t, headers.Metadata().Get("Foo"), "bar")
	expectString(t, headers.Metadata().Get("FOO"), "bar")
}

// TODO TestParseAccountHeadersError
