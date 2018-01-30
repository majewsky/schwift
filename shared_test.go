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
	"os"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/swauth"
)

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
