// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package schwift

import (
	"bytes"
	"io"
	"testing"
)

func TestParseHTTPRange(t *testing.T) {
	testCases := []struct {
		input  string
		ok     bool
		offset int64
		length uint64
	}{
		// all the testcases from RFC 7233, section 3.1
		{"0-499", true, 0, 500},
		{"500-999", true, 500, 500},
		{"-500", true, -1, 500},
		{"9500-", true, 9500, 0},
		{"0-0", true, 0, 1},
		{"-1", true, -1, 1},
		// and then some more
		{"0-", true, 0, 0},
		{"-", true, 0, 0},
		// some error cases for 100% coverage
		{"no dash", false, 0, 0},
		{"what-the-heck", false, 0, 0},
		{"-X", false, 0, 0},
		{"X-X", false, 0, 0},
		{"X-", false, 0, 0},
		{"999-500", false, 0, 0},
	}

	for _, tc := range testCases {
		o, l, ok := parseHTTPRange(tc.input)

		if tc.ok && !ok {
			t.Errorf("expected %q to parse, but did not", tc.input)
		}
		if !tc.ok && ok {
			t.Errorf("expected %q to fail, but parsed into (%d, %d)",
				tc.input, o, l)
		}
		if o != tc.offset || l != tc.length {
			t.Errorf("expected %q to parse as (%d, %d), but (%d, %d)",
				tc.input, tc.offset, tc.length, o, l)
		}
	}
}

func TestSegmentingReader(t *testing.T) {
	testCases := []struct {
		input    string
		segments []string
	}{
		{"abcdefghi", []string{"abc", "def", "ghi"}},
		{"abcdefgh", []string{"abc", "def", "gh"}},
		{"abcdefg", []string{"abc", "def", "g"}},
	}

	for _, tc := range testCases {
		sr := segmentingReader{
			Reader:           bytes.NewReader([]byte(tc.input)),
			SegmentSizeBytes: 3,
		}

		for _, expected := range tc.segments {
			segment := sr.NextSegment()
			if segment == nil {
				t.Errorf("expected segment %q, but NextSegment() returned nil", expected)
				break
			}
			actual, err := io.ReadAll(segment)
			if err != nil {
				t.Errorf("expected segment %q, but got read error %q", expected, err.Error())
				break
			}
			if string(actual) != expected {
				t.Errorf("expected segment %q, but got %q", expected, string(actual))
			}
		}

		segment := sr.NextSegment()
		if segment != nil {
			actual, err := io.ReadAll(segment)
			if err != nil {
				t.Errorf("expected no more segments, but got segment producing read error %q", err.Error())
			} else {
				t.Errorf("expected no more segments, but got %q", string(actual))
			}
		}
	}
}
