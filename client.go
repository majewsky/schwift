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
	"net/http"

	"github.com/gophercloud/gophercloud"
)

//Client is the interface between Schwift and the libraries providing
//authentication for it. Schwift can wrap gophercloud.ServiceClient to provide
//this interface, so if you have a gophercloud.ServiceClient, use the
//AccountFromGophercloud method to obtain the corresponding schwift.Account
//instance.
type Client interface {
	//EndpointURL returns the endpoint URL from the Keystone catalog for the
	//Swift account that this client operates on. It should look like
	//`http://domain.tld/v1/AUTH_projectid/`.
	EndpointURL() string
	//Clone returns a deep clone of this client with the endpoint URL changed to
	//the given URL.
	Clone(newEndpointURL string) Client
	//Do executes the given request after adding to it the X-Auth-Token header
	//containing the client's current Keystone (or Swift auth) token. It may
	//also set other headers, such as User-Agent. If the status code returned is
	//401, it shall attempt to acquire a new auth token and restart the request
	//with the new token.
	Do(req *http.Request) (*http.Response, error)
}

type gophercloudClient struct {
	c *gophercloud.ServiceClient
}

func (g *gophercloudClient) EndpointURL() string {
	return g.c.Endpoint
}

func (g *gophercloudClient) Clone(newEndpointURL string) Client {
	clonedClient := *g.c
	clonedClient.Endpoint = newEndpointURL
	return &gophercloudClient{&clonedClient}
}

func (g *gophercloudClient) Do(req *http.Request) (*http.Response, error) {
	return g.do(req, false)
}

func (g *gophercloudClient) do(req *http.Request, afterReauth bool) (*http.Response, error) {
	provider := g.c.ProviderClient

	req.Header.Set("User-Agent", provider.UserAgent.Join())
	for key, value := range provider.AuthenticatedHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := provider.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	//detect expired token
	if resp.StatusCode == http.StatusUnauthorized && !afterReauth {
		err := drainResponseBody(resp)
		if err != nil {
			return nil, err
		}
		err = provider.Reauthenticate(resp.Request.Header.Get("X-Auth-Token"))
		if err != nil {
			return nil, err
		}
		//restart request with new token
		return g.do(req, true)
	}

	return resp, nil
}
