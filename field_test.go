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
	"strconv"
	"testing"
)

func TestFieldString(t *testing.T) {
	hdr := make(AccountHeaders)
	expectBool(t, hdr.TempURLKey().Exists(), false)
	expectString(t, hdr.TempURLKey().Get(), "")
	expectError(t, hdr.Validate(), "")

	hdr["X-Account-Meta-Temp-Url-Key"] = ""
	expectBool(t, hdr.TempURLKey().Exists(), false)
	expectString(t, hdr.TempURLKey().Get(), "")
	expectError(t, hdr.Validate(), "")

	hdr["X-Account-Meta-Temp-Url-Key"] = "foo"
	expectBool(t, hdr.TempURLKey().Exists(), true)
	expectString(t, hdr.TempURLKey().Get(), "foo")
	expectError(t, hdr.Validate(), "")

	hdr.TempURLKey().Set("bar")
	expectHeaders(t, hdr, map[string]string{
		"X-Account-Meta-Temp-Url-Key": "bar",
	})
	hdr.TempURLKey().Clear()
	expectHeaders(t, hdr, map[string]string{
		"X-Account-Meta-Temp-Url-Key": "",
	})
	hdr.TempURLKey().Del()
	expectHeaders(t, hdr, nil)
	hdr.TempURLKey().Clear()
	expectHeaders(t, hdr, map[string]string{
		"X-Account-Meta-Temp-Url-Key": "",
	})
}

////////////////////////////////////////////////////////////////////////////////

func TestFieldTimestamp(t *testing.T) {
	testWithAccount(t, func(a *Account) {
		hdr, err := a.Headers()
		if !expectError(t, err, "") {
			return
		}

		expectBool(t, hdr.Timestamp().Exists(), true)

		actual := float64(hdr.Timestamp().Get().UnixNano()) / 1e9
		expected, _ := strconv.ParseFloat(hdr["X-Timestamp"], 64)
		expectFloat64(t, actual, expected)
	})

	hdr := make(AccountHeaders)
	expectBool(t, hdr.Timestamp().Exists(), false)
	expectBool(t, hdr.Timestamp().Get().IsZero(), true)
	expectError(t, hdr.Validate(), "")

	hdr["X-Timestamp"] = "wtf"
	expectBool(t, hdr.Timestamp().Exists(), true)
	expectBool(t, hdr.Timestamp().Get().IsZero(), true)
	expectError(t, hdr.Validate(), `Bad header X-Timestamp: strconv.ParseFloat: parsing "wtf": invalid syntax`)
}

////////////////////////////////////////////////////////////////////////////////

func TestFieldUint64(t *testing.T) {
	hdr := make(AccountHeaders)
	expectBool(t, hdr.BytesUsedQuota().Exists(), false)
	expectUint64(t, hdr.BytesUsedQuota().Get(), 0)
	expectError(t, hdr.Validate(), "")

	hdr["X-Account-Meta-Quota-Bytes"] = "23"
	expectBool(t, hdr.BytesUsedQuota().Exists(), true)
	expectUint64(t, hdr.BytesUsedQuota().Get(), 23)
	expectError(t, hdr.Validate(), "")

	hdr["X-Account-Meta-Quota-Bytes"] = "-23"
	expectBool(t, hdr.BytesUsedQuota().Exists(), true)
	expectUint64(t, hdr.BytesUsedQuota().Get(), 0)
	expectError(t, hdr.Validate(), `Bad header X-Account-Meta-Quota-Bytes: strconv.ParseUint: parsing "-23": invalid syntax`)

	hdr.BytesUsedQuota().Set(9001)
	expectHeaders(t, hdr, map[string]string{
		"X-Account-Meta-Quota-Bytes": "9001",
	})
	hdr.BytesUsedQuota().Clear()
	expectHeaders(t, hdr, map[string]string{
		"X-Account-Meta-Quota-Bytes": "",
	})
	hdr.BytesUsedQuota().Del()
	expectHeaders(t, hdr, nil)
	hdr.BytesUsedQuota().Clear()
	expectHeaders(t, hdr, map[string]string{
		"X-Account-Meta-Quota-Bytes": "",
	})
}

func TestFieldUint64Readonly(t *testing.T) {
	hdr := make(AccountHeaders)
	expectBool(t, hdr.BytesUsed().Exists(), false)
	expectUint64(t, hdr.BytesUsed().Get(), 0)
	expectError(t, hdr.Validate(), "")

	hdr["X-Account-Bytes-Used"] = "23"
	expectBool(t, hdr.BytesUsed().Exists(), true)
	expectUint64(t, hdr.BytesUsed().Get(), 23)
	expectError(t, hdr.Validate(), "")

	hdr["X-Account-Bytes-Used"] = "-23"
	expectBool(t, hdr.BytesUsed().Exists(), true)
	expectUint64(t, hdr.BytesUsed().Get(), 0)
	expectError(t, hdr.Validate(), `Bad header X-Account-Bytes-Used: strconv.ParseUint: parsing "-23": invalid syntax`)
}
