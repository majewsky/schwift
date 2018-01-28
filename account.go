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
	"fmt"
	"net/http"
	"regexp"

	"github.com/gophercloud/gophercloud"
)

//Account represents a Swift account.
type Account struct {
	client *gophercloud.ServiceClient
	//URL parts
	baseURL string
	name    string
	//cache
	metadata *AccountMetadata
}

////////////////////////////////////////////////////////////////////////////////
// interface to Gophercloud, endpoint inspection and manipulation

var endpointURLRegexp = regexp.MustCompile(`^(.*/)v1/(.*)/$`)

//AccountFromClient takes a gophercloud.ServiceClient which wraps a Swift
//endpoint, and returns the Account instance corresponding to the account or
//project that this client is connected to.
func AccountFromClient(client *gophercloud.ServiceClient) (*Account, error) {
	match := endpointURLRegexp.FindStringSubmatch(client.Endpoint)
	if match == nil {
		return nil, fmt.Errorf(`schwift.AccountFromClient(): invalid Swift endpoint URL: cannot find "/v1/" in %q`, client.Endpoint)
	}
	return &Account{
		client:  client,
		baseURL: match[1],
		name:    match[2],
	}, nil
}

//SwitchAccount returns a handle on a different account on the same server. Note
//that you need reseller permissions to access accounts other than that where
//you originally authenticated. This method does not check whether the account
//actually exists.
//
//The account name is usually the project name with an additional "AUTH_"
//prefix.
func (a *Account) SwitchAccount(accountName string) *Account {
	clonedClient := *a.client
	clonedClient.Endpoint = a.baseURL + "v1/" + accountName + "/"
	return &Account{
		client:  &clonedClient,
		baseURL: a.baseURL,
		name:    accountName,
	}
}

//Name returns the name of the account (usually the prefix "AUTH_" followed by
//the project ID).
func (a *Account) Name() string {
	return a.name
}

//Client returns the gophercloud.ServiceClient which is used to make requests
//against this account.
func (a *Account) Client() *gophercloud.ServiceClient {
	return a.client
}

////////////////////////////////////////////////////////////////////////////////
// account metadata

//AccountMetadata contains the metadata for an account. The `Raw` attribute
//contains the raw set of headers returned from a HEAD or GET request on the
//account. The other attributes contain the parsed values of common headers.
type AccountMetadata struct {
	Exists         bool
	BytesUsed      uint64 //from X-Account-Bytes-Used
	ContainerCount uint64 //from X-Account-Container-Count
	ObjectCount    uint64 //from X-Account-Object-Count
	//TODO account quota
	Raw http.Header
}

//Metadata returns the metadata for this account. If the account does not exist,
func (a *Account) Metadata() (AccountMetadata, error) {
	if a.metadata != nil {
		return *a.metadata, nil
	}

	resp, err := Request{
		Method:            "HEAD",
		ExpectStatusCodes: []int{200},
	}.Do(a.client)
	if err != nil {
		return AccountMetadata{}, err
	}

	a.metadata, err = parseAccountMetadata(resp)
	if err != nil {
		return AccountMetadata{}, err
	}
	return *a.metadata, nil
}

func parseAccountMetadata(resp *http.Response) (*AccountMetadata, error) {
	bytesUsed, err := parseUnsignedIntHeader(resp, "X-Account-Bytes-Used")
	if err != nil {
		return nil, err
	}
	containerCount, err := parseUnsignedIntHeader(resp, "X-Account-Container-Count")
	if err != nil {
		return nil, err
	}
	objectCount, err := parseUnsignedIntHeader(resp, "X-Account-Object-Count")
	if err != nil {
		return nil, err
	}
	return &AccountMetadata{
		Exists:         true,
		BytesUsed:      bytesUsed,
		ContainerCount: containerCount,
		ObjectCount:    objectCount,
		Raw:            resp.Header,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// container listing
