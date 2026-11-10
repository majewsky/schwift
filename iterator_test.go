// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package schwift

import (
	"testing"
	"time"
)

func TestParseSwiftLastModified(t *testing.T) {
	want := time.Date(2026, time.November, 10, 15, 37, 37, 707_000_000, time.UTC)
	for _, input := range []string{
		"2026-11-10T15:37:37.707",       // Swift's timezone-less format
		"2026-11-10T15:37:37.707Z",      // UTC timezone
		"2026-11-10T16:37:37.707+01:00", // Explicit offset
	} {
		t.Run(input, func(t *testing.T) {
			got, err := parseSwiftLastModified(input)
			if err != nil {
				t.Fatalf("parseSwiftLastModified(%q) returned an error: %v", input, err)
			}
			if !got.Equal(want) {
				t.Errorf("parseSwiftLastModified(%q) = %s, want %s", input, got, want)
			}
		})
	}
}
