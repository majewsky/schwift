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

import "testing"

func TestHeaders(t *testing.T) {
	h := make(Headers)
	h.Set("first", "value1")
	h.Set("second-thing", "value2")

	expectHeaders(t, h, map[string]string{
		"First":        "value1",
		"Second-Thing": "value2",
	})

	expectString(t, h.Get("first"), "value1")
	expectString(t, h.Get("First"), "value1")
	expectString(t, h.Get("FIRST"), "value1")

	h.Set("first", "changed")
	h.Set("third", "")

	expectHeaders(t, h, map[string]string{
		"First":        "changed",
		"Second-Thing": "value2",
		"Third":        "",
	})

	h.Clear("second-thing")
	h.Clear("fourth-thing")

	expectHeaders(t, h, map[string]string{
		"First":        "changed",
		"Second-Thing": "",
		"Third":        "",
		"Fourth-Thing": "",
	})

	h.Del("FIRST")
	h.Del("second-Thing")

	expectHeaders(t, h, map[string]string{
		"Third":        "",
		"Fourth-Thing": "",
	})

}

func expectString(t *testing.T, actual string, expected string) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected value %q, got %q instead\n", expected, actual)
	}
}

func expectHeaders(t *testing.T, actual Headers, expected map[string]string) {
	t.Helper()
	reported := make(map[string]bool)

	for k, av := range actual {
		ev, exists := expected[k]
		if !exists {
			ev = "<not set>"
		}
		if av != ev {
			t.Errorf(`expected "%s: %s", got "%s: %s" instead`, k, ev, k, av)
			reported[k] = true
		}
	}

	for k, ev := range expected {
		av, exists := actual[k]
		if !exists {
			av = "<not set>"
		}
		if av != ev && !reported[k] {
			t.Errorf(`expected "%s: %s", got "%s: %s" instead`, k, ev, k, av)
		}
	}
}
