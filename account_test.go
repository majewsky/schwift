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
	"testing"
)

func TestAccountBasic(t *testing.T) {
	testWithAccount(t, func(a *Account) {
		hdr, err := a.Headers()
		if !expectError(t, err, "") {
			t.FailNow()
		}
		//There are not a lot of things we can test here (besides testing that
		//Headers() does not fail, i.e. everything parses correctly), but
		//Content-Type is going to be text/plain because GET on an account lists
		//the container names as plain text.
		expectString(t, hdr.Raw.Get("Content-Type"), "text/plain; charset=utf-8")
	})
}

func TestAccountMetadata(t *testing.T) {
	testWithAccount(t, func(a *Account) {
		//test creating some metadata
		err := a.Update(AccountHeaders{
			Metadata: NewMetadata(
				"schwift-test1", "first",
				"schwift-test2", "second",
			),
		}, nil)
		if !expectError(t, err, "") {
			t.FailNow()
		}

		hdr, err := a.Headers()
		if !expectError(t, err, "") {
			t.FailNow()
		}
		expectString(t, hdr.Metadata.Get("schwift-test1"), "first")
		expectString(t, hdr.Metadata.Get("schwift-test2"), "second")

		//test deleting some metadata
		m := make(Metadata)
		m.Clear("schwift-test1")
		err = a.Update(AccountHeaders{
			Metadata: m,
		}, nil)
		if !expectError(t, err, "") {
			t.FailNow()
		}

		hdr, err = a.Headers()
		if !expectError(t, err, "") {
			t.FailNow()
		}
		expectString(t, hdr.Metadata.Get("schwift-test1"), "")
		expectString(t, hdr.Metadata.Get("schwift-test2"), "second")

		//test updating some metadata
		m = make(Metadata)
		m.Set("schwift-test2", "changed")
		err = a.Update(AccountHeaders{
			Metadata: m,
		}, nil)
		if !expectError(t, err, "") {
			t.FailNow()
		}

		hdr, err = a.Headers()
		if !expectError(t, err, "") {
			t.FailNow()
		}
		expectString(t, hdr.Metadata.Get("schwift-test1"), "")
		expectString(t, hdr.Metadata.Get("schwift-test2"), "changed")

	})
}
