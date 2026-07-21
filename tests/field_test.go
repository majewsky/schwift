// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"net/http"
	"strconv"
	"testing"

	"go.xyrillian.de/schwift/v2"
)

func TestFieldString(t *testing.T) {
	hdr := schwift.NewAccountHeaders()
	expectBool(t, hdr.TempURLKey().Exists(), false)
	expectString(t, hdr.TempURLKey().Get(), "")
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Account-Meta-Temp-Url-Key"] = ""
	expectBool(t, hdr.TempURLKey().Exists(), false)
	expectString(t, hdr.TempURLKey().Get(), "")
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Account-Meta-Temp-Url-Key"] = "foo"
	expectBool(t, hdr.TempURLKey().Exists(), true)
	expectString(t, hdr.TempURLKey().Get(), "foo")
	expectSuccess(t, hdr.Validate())

	hdr.TempURLKey().Set("bar")
	expectHeaders(t, hdr.Headers, map[string]string{
		"X-Account-Meta-Temp-Url-Key": "bar",
	})
	hdr.TempURLKey().Clear()
	expectHeaders(t, hdr.Headers, map[string]string{
		"X-Account-Meta-Temp-Url-Key": "",
	})
	hdr.TempURLKey().Del()
	expectHeaders(t, hdr.Headers, nil)
	hdr.TempURLKey().Clear()
	expectHeaders(t, hdr.Headers, map[string]string{
		"X-Account-Meta-Temp-Url-Key": "",
	})
}

////////////////////////////////////////////////////////////////////////////////

func TestFieldTimestamp(t *testing.T) {
	testWithAccount(t, func(a *schwift.Account) {
		hdr, err := a.Headers(t.Context())
		if !expectSuccess(t, err) {
			return
		}

		expectBool(t, hdr.CreatedAt().Exists(), true)

		actual := float64(hdr.CreatedAt().Get().UnixNano()) / 1e9
		expected, _ := strconv.ParseFloat(hdr.Headers["X-Timestamp"], 64) //nolint:errcheck
		expectFloat64(t, actual, expected)
	})

	hdr := schwift.NewAccountHeaders()
	expectBool(t, hdr.CreatedAt().Exists(), false)
	expectBool(t, hdr.CreatedAt().Get().IsZero(), true)
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Timestamp"] = "wtf"
	expectBool(t, hdr.CreatedAt().Exists(), true)
	expectBool(t, hdr.CreatedAt().Get().IsZero(), true)
	expectError(t, hdr.Validate(), `Bad header X-Timestamp: strconv.ParseFloat: parsing "wtf": invalid syntax`)
}

func TestFieldHTTPTimestamp(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		obj := c.Object("test")
		err := obj.Upload(t.Context(), nil, nil, nil)
		if !expectSuccess(t, err) {
			return
		}

		hdr, err := obj.Headers(t.Context())
		if !expectSuccess(t, err) {
			return
		}
		expectBool(t, hdr.UpdatedAt().Exists(), true)

		actual := hdr.UpdatedAt().Get()
		expected, _ := http.ParseTime(hdr.Get("Last-Modified")) //nolint:errcheck
		expectInt64(t, actual.Unix(), expected.Unix())
	})

	hdr := schwift.NewObjectHeaders()
	expectBool(t, hdr.UpdatedAt().Exists(), false)
	expectBool(t, hdr.UpdatedAt().Get().IsZero(), true)
	expectSuccess(t, hdr.Validate())

	hdr.Headers["Last-Modified"] = "wtf"
	expectBool(t, hdr.UpdatedAt().Exists(), true)
	expectBool(t, hdr.UpdatedAt().Get().IsZero(), true)
	expectError(t, hdr.Validate(), `Bad header Last-Modified: parsing time "wtf" as "Mon Jan _2 15:04:05 2006": cannot parse "wtf" as "Mon"`)
}

////////////////////////////////////////////////////////////////////////////////

func TestFieldUint64(t *testing.T) {
	hdr := schwift.NewAccountHeaders()
	expectBool(t, hdr.BytesUsedQuota().Exists(), false)
	expectUint64(t, hdr.BytesUsedQuota().Get(), 0)
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Account-Meta-Quota-Bytes"] = "23"
	expectBool(t, hdr.BytesUsedQuota().Exists(), true)
	expectUint64(t, hdr.BytesUsedQuota().Get(), 23)
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Account-Meta-Quota-Bytes"] = "-23"
	expectBool(t, hdr.BytesUsedQuota().Exists(), true)
	expectUint64(t, hdr.BytesUsedQuota().Get(), 0)
	expectError(t, hdr.Validate(), `Bad header X-Account-Meta-Quota-Bytes: strconv.ParseUint: parsing "-23": invalid syntax`)

	hdr.BytesUsedQuota().Set(9001)
	expectHeaders(t, hdr.Headers, map[string]string{
		"X-Account-Meta-Quota-Bytes": "9001",
	})
	hdr.BytesUsedQuota().Clear()
	expectHeaders(t, hdr.Headers, map[string]string{
		"X-Account-Meta-Quota-Bytes": "",
	})
	hdr.BytesUsedQuota().Del()
	expectHeaders(t, hdr.Headers, nil)
	hdr.BytesUsedQuota().Clear()
	expectHeaders(t, hdr.Headers, map[string]string{
		"X-Account-Meta-Quota-Bytes": "",
	})
}

func TestFieldUint64Readonly(t *testing.T) {
	hdr := schwift.NewAccountHeaders()
	expectBool(t, hdr.BytesUsed().Exists(), false)
	expectUint64(t, hdr.BytesUsed().Get(), 0)
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Account-Bytes-Used"] = "23"
	expectBool(t, hdr.BytesUsed().Exists(), true)
	expectUint64(t, hdr.BytesUsed().Get(), 23)
	expectSuccess(t, hdr.Validate())

	hdr.Headers["X-Account-Bytes-Used"] = "-23"
	expectBool(t, hdr.BytesUsed().Exists(), true)
	expectUint64(t, hdr.BytesUsed().Get(), 0)
	expectError(t, hdr.Validate(), `Bad header X-Account-Bytes-Used: strconv.ParseUint: parsing "-23": invalid syntax`)
}
