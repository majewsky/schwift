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
	headers *AccountHeaders
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
// account headers

//Headers returns the AccountHeaders for this account. If the AccountHeaders
//has not been cached yet, a HEAD request is issued on the account.
func (a *Account) Headers() (AccountHeaders, error) {
	if a.headers != nil {
		return *a.headers, nil
	}

	resp, err := Request{
		Method:            "HEAD",
		ExpectStatusCodes: []int{200},
	}.Do(a.client)
	if err != nil {
		return AccountHeaders{}, err
	}

	var headers AccountHeaders
	err = parseHeaders(resp.Header, &headers)
	if err != nil {
		return AccountHeaders{}, err
	}
	return *a.headers, nil
}

//Invalidate clears the internal cache of this Account instance. The next call
//to Headers() on this instance will issue a HEAD request on the account.
func (a *Account) Invalidate() {
	a.headers = nil
}

//Post creates or updates the account using a POST request.
func (a *Account) Post(headers AccountHeaders) error {
	_, err := Request{
		Method:            "POST",
		AdditionalHeaders: compileHeaders(headers),
		ExpectStatusCodes: []int{204},
	}.Do(a.client)
	return err
}

////////////////////////////////////////////////////////////////////////////////
// container listing
