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
	"io"
	"io/ioutil"
	"net/http"
	urlmodule "net/url"
	"strings"

	"github.com/gophercloud/gophercloud"
)

var okCodes []int

func init() {
	//prepare input for gophercloud.RequestOpts.OkCodes such that gophercloud's
	//error handling is fused
	for code := 100; code < 600; code++ {
		//as an exception, 401s are handled by Gophercloud because we want to use its
		//internal token renewal logic
		if code != 401 {
			okCodes = append(okCodes, code)
		}
	}
}

//Request contains the parameters that can be set in a request to the Swift API.
type Request struct {
	Method            string //"GET", "HEAD", "PUT", "POST" or "DELETE"
	ContainerName     string //empty for requests on accounts
	ObjectName        string //empty for requests on accounts/containers
	AdditionalHeaders map[string]string
	//ExpectStatusCodes can be left empty to disable this check, otherwise
	//schwift.UnexpectedStatusCodeError may be returned.
	ExpectStatusCodes []int
}

//URL returns the full URL for this request.
func (r Request) URL(client *gophercloud.ServiceClient) (string, error) {
	url, err := urlmodule.Parse(client.Endpoint)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(url.Path, "/") {
		url.Path += "/"
	}

	if r.ContainerName == "" {
		if r.ObjectName != "" {
			return "", ErrNoContainerName
		}
	} else {
		if strings.Contains(r.ContainerName, "/") {
			return "", ErrMalformedContainerName
		}
		url.Path += r.ContainerName + "/" + r.ObjectName
	}

	return url.String(), nil
}

//Do executes this request on the given service client.
func (r Request) Do(client *gophercloud.ServiceClient) (*http.Response, error) {
	return r.do(client, false)
}

func (r Request) do(client *gophercloud.ServiceClient, afterReauth bool) (*http.Response, error) {
	//build URL
	url, err := r.URL(client)
	if err != nil {
		return nil, err
	}

	//override gophercloud's error handling
	opts := &gophercloud.RequestOpts{OkCodes: okCodes}

	//override gophercloud's default headers
	opts.MoreHeaders = map[string]string{
		"Accept":       "",
		"Content-Type": "",
	}
	for key, value := range r.AdditionalHeaders {
		opts.MoreHeaders[key] = value
	}

	resp, err := client.ProviderClient.Request(r.Method, url, opts)
	if err != nil {
		return resp, err
	}

	//return success if error code matches expectation
	if len(r.ExpectStatusCodes) == 0 {
		//check disabled -> return response unaltered
		return resp, nil
	}
	for _, code := range r.ExpectStatusCodes {
		if code == resp.StatusCode {
			return resp, nil
		}
	}

	//since we override gophercloud's error handling, we need to handle token
	//expiry ourselves
	if resp.StatusCode == http.StatusUnauthorized && !afterReauth {
		err := drainResponseBody(resp)
		if err != nil {
			return nil, err
		}
		err = client.Reauthenticate(resp.Request.Header.Get("X-Auth-Token"))
		if err != nil {
			return nil, err
		}
		//restart request with new token
		return r.do(client, true)
	}

	//other unexpected status code -> generate UnexpectedStatusCodeError
	buf, err := collectResponseBody(resp)
	if err != nil {
		return nil, err
	}
	return nil, UnexpectedStatusCodeError{
		ExpectedStatusCodes: r.ExpectStatusCodes,
		ActualResponse:      resp,
		ResponseBody:        buf,
	}
}

func drainResponseBody(r *http.Response) error {
	_, err := io.Copy(ioutil.Discard, r.Body)
	if err != nil {
		return err
	}
	return r.Body.Close()
}

func collectResponseBody(r *http.Response) ([]byte, error) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return buf, r.Body.Close()
}
