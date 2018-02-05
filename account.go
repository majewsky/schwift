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

//Headers returns the AccountHeaders for this account. If the AccountHeaders
//has not been cached yet, a HEAD request is issued on the account.
func (a *Account) Headers() (AccountHeaders, error) {
	if a.headers != nil {
		return *a.headers, nil
	}

	resp, err := Request{
		Method:            "HEAD",
		ExpectStatusCodes: []int{204},
	}.Do(a.client)
	if err != nil {
		return AccountHeaders{}, err
	}

	headers := NewAccountHeaders()
	headers.FromHTTP(resp.Header)
	err = headers.Validate()
	if err != nil {
		return headers, err
	}
	a.headers = &headers
	return *a.headers, nil
}

//Invalidate clears the internal cache of this Account instance. The next call
//to Headers() on this instance will issue a HEAD request on the account.
func (a *Account) Invalidate() {
	a.headers = nil
}

//Update updates the account using a POST request. To add URL parameters, pass
//a non-nil *RequestOptions.
//
//A successful POST request implies Invalidate() since it may change metadata.
func (a *Account) Update(headers AccountHeaders, opts *RequestOptions) error {
	_, err := Request{
		Method:            "POST",
		Headers:           headers.ToHTTP(),
		Options:           opts,
		ExpectStatusCodes: []int{204},
	}.Do(a.client)
	if err == nil {
		a.Invalidate()
	}
	return err
}

//Create creates the account using a PUT request. To add URL parameters, pass
//a non-nil *RequestOptions.
//
//Note that this operation is only available to reseller admins, not to regular
//users.
//
//A successful PUT request implies Invalidate() since it may change metadata.
func (a *Account) Create(headers AccountHeaders, opts *RequestOptions) error {
	_, err := Request{
		Method:            "PUT",
		Headers:           headers.ToHTTP(),
		Options:           opts,
		ExpectStatusCodes: []int{201, 202},
	}.Do(a.client)
	if err == nil {
		a.Invalidate()
	}
	return err
}

// TODO container listing
