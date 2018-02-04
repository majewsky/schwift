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

import "testing"

func TestMetadata(t *testing.T) {
	m := NewMetadata(
		"first", "value1",
		"second-thing", "value2",
	)

	expectMetadata(t, m, map[string]string{
		"First":        "value1",
		"Second-Thing": "value2",
	})

	expectString(t, m.Get("first"), "value1")
	expectString(t, m.Get("First"), "value1")
	expectString(t, m.Get("FIRST"), "value1")

	m.Set("first", "changed")
	m.Set("third", "")

	expectMetadata(t, m, map[string]string{
		"First":        "changed",
		"Second-Thing": "value2",
		"Third":        "",
	})

	m.Clear("second-thing")
	m.Clear("fourth-thing")

	expectMetadata(t, m, map[string]string{
		"First":        "changed",
		"Second-Thing": "",
		"Third":        "",
		"Fourth-Thing": "",
	})

	m.Del("FIRST")
	m.Del("second-Thing")

	expectMetadata(t, m, map[string]string{
		"Third":        "",
		"Fourth-Thing": "",
	})

}

func expectMetadata(t *testing.T, actual Metadata, expected map[string]string) {
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
