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
	"crypto/rand"
	"encoding/hex"
	"math"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/swauth"
)

//This function can be used during debugging to redirect the HTTP requests for
//a specific unit test through a mitmproxy. Put this at the beginning of your
//testcase like so:
//
//	testWithAccount(t, func(a *Account) {
//	    withProxy(a, "http://localhost:8888", func() {
//	        ...
//
//	testWithContainer(t, func(c *Container) {
//	    withProxy(c.Account(), "http://localhost:8888", func() {
//	        ...
func withProxy(a *Account, proxyURL string, action func()) {
	t := http.DefaultTransport.(*http.Transport)
	proxyOrig := t.Proxy
	t.Proxy = func(*http.Request) (*url.URL, error) { return url.Parse(proxyURL) }
	a.client.ProviderClient.HTTPClient.Transport = t
	action()
	t.Proxy = proxyOrig
}

func testWithAccount(t *testing.T, testCode func(a *Account)) {
	stAuth := os.Getenv("ST_AUTH")
	stUser := os.Getenv("ST_USER")
	stKey := os.Getenv("ST_KEY")
	var client *gophercloud.ServiceClient

	if stAuth == "" && stUser == "" && stKey == "" {
		//option 1: Keystone authentication
		authOptions, err := openstack.AuthOptionsFromEnv()
		if err != nil {
			t.Error("missing Swift credentials (need either ST_AUTH, ST_USER, ST_KEY or OS_* variables)")
			t.Error("openstack.AuthOptionsFromEnv returned: " + err.Error())
			return
		}
		provider, err := openstack.AuthenticatedClient(authOptions)
		if err != nil {
			t.Errorf("openstack.AuthenticatedClient returned: " + err.Error())
			return
		}
		client, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
		if err != nil {
			t.Errorf("openstack.NewObjectStorageV1 returned: " + err.Error())
			return
		}
	} else {
		//option 2: Swift authentication v1
		provider, err := openstack.NewClient(stAuth)
		if err != nil {
			t.Errorf("openstack.NewClient returned: " + err.Error())
			return
		}
		client, err = swauth.NewObjectStorageV1(provider, swauth.AuthOpts{User: stUser, Key: stKey})
		if err != nil {
			t.Errorf("swauth.NewObjectStorageV1 returned: " + err.Error())
			return
		}
	}

	account, err := AccountFromClient(client)
	if err != nil {
		t.Errorf("schwift.AccountFromClient returned: " + err.Error())
		return
	}
	testCode(account)
}

func testWithContainer(t *testing.T, testCode func(c *Container)) {
	testWithAccount(t, func(a *Account) {
		containerName := getRandomName()
		container, err := a.Container(containerName).EnsureExists()
		expectError(t, err, "")

		testCode(container)

		//cleanup
		exists, err := container.Exists()
		expectError(t, err, "")
		if exists {
			err = container.Delete(nil, nil)
			expectError(t, err, "")
		}
	})
}

////////////////////////////////////////////////////////////////////////////////

func getRandomName() string {
	var buf [16]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err.Error())
	}
	return hex.EncodeToString(buf[:])
}

////////////////////////////////////////////////////////////////////////////////

func expectBool(t *testing.T, actual bool, expected bool) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected value %#v, got %#v instead\n", expected, actual)
	}
}

func expectFloat64(t *testing.T, actual float64, expected float64) {
	t.Helper()
	if math.Abs((actual-expected)/expected) > 1e-8 {
		t.Errorf("expected value %g, got %g instead\n", expected, actual)
	}
}

func expectInt(t *testing.T, actual int, expected int) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected value %d, got %d instead\n", expected, actual)
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

func expectHeaders(t *testing.T, actual map[string]string, expected map[string]string) {
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

func expectSuccess(t *testing.T, actual error) (ok bool) {
	return expectError(t, actual, "")
}
