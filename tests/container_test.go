// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"net/http"
	"testing"

	"go.xyrillian.de/schwift/v2"
)

func TestContainerLifecycle(t *testing.T) {
	testWithAccount(t, func(a *schwift.Account) {
		containerName := getRandomName()
		c := a.Container(containerName)

		expectString(t, c.Name(), containerName)
		if c.Account() != a {
			t.Errorf("expected c.Account() = %#v, got %#v instead\n", a, c.Account())
		}

		exists, err := c.Exists(t.Context())
		expectSuccess(t, err)
		expectBool(t, exists, false)

		_, err = c.Headers(t.Context())
		expectError(t, err, fmt.Sprintf("could not HEAD %q in Swift: expected 204 response, got 404 instead", containerName))
		expectBool(t, schwift.Is(err, http.StatusNotFound), true)
		expectBool(t, schwift.Is(err, http.StatusNoContent), false)

		// DELETE should be idempotent and not return success on non-existence, but
		// OpenStack LOVES to be inconsistent with everything (including, notably, itself)
		err = c.Delete(t.Context(), nil)
		expectError(t, err, fmt.Sprintf("could not DELETE %q in Swift: expected 204 response, got 404 instead: <html><h1>Not Found</h1><p>The resource could not be found.</p></html>", containerName))

		err = c.Create(t.Context(), nil)
		expectSuccess(t, err)

		exists, err = c.Exists(t.Context())
		expectSuccess(t, err)
		expectBool(t, exists, true)

		err = c.Delete(t.Context(), nil)
		expectSuccess(t, err)
	})
}

func TestContainerUpdate(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		hdr, err := c.Headers(t.Context())
		expectSuccess(t, err)
		expectBool(t, hdr.ObjectCount().Exists(), true)
		expectUint64(t, hdr.ObjectCount().Get(), 0)

		hdr = schwift.NewContainerHeaders()
		hdr.ObjectCountQuota().Set(23)
		hdr.BytesUsedQuota().Set(42)

		err = c.Update(t.Context(), hdr, nil)
		expectSuccess(t, err)

		hdr, err = c.Headers(t.Context())
		expectSuccess(t, err)
		expectUint64(t, hdr.BytesUsedQuota().Get(), 42)
		expectUint64(t, hdr.ObjectCountQuota().Get(), 23)
	})
}

func expectContainerExistence(t *testing.T, c *schwift.Container, expectedExists bool) {
	t.Helper()
	c.Invalidate()
	actualExists, err := c.Exists(t.Context())
	expectSuccess(t, err)
	expectBool(t, actualExists, expectedExists)
}
