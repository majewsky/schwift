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
	"net/url"
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
	Method        string //"GET", "HEAD", "PUT", "POST" or "DELETE"
	ContainerName string //empty for requests on accounts
	ObjectName    string //empty for requests on accounts/containers
	Options       RequestOptions
	Body          io.Reader
	//ExpectStatusCodes can be left empty to disable this check, otherwise
	//schwift.UnexpectedStatusCodeError may be returned.
	ExpectStatusCodes []int
}

//RequestOptions contains additional headers and values for request.
type RequestOptions struct {
	Headers http.Header
	Values  url.Values
}

//URL returns the full URL for this request.
func (r Request) URL(client *gophercloud.ServiceClient, values url.Values) (string, error) {
	uri, err := url.Parse(client.Endpoint)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(uri.Path, "/") {
		uri.Path += "/"
	}

	if r.ContainerName == "" {
		if r.ObjectName != "" {
			return "", ErrNoContainerName
		}
	} else {
		if strings.Contains(r.ContainerName, "/") {
			return "", ErrMalformedContainerName
		}
		uri.Path += r.ContainerName + "/" + r.ObjectName
	}

	uri.RawQuery = values.Encode()
	return uri.String(), nil
}

//Do executes this request on the given service client.
func (r Request) Do(client *gophercloud.ServiceClient) (*http.Response, error) {
	return r.do(client, false)
}

func (r Request) do(client *gophercloud.ServiceClient, afterReauth bool) (*http.Response, error) {
	provider := client.ProviderClient

	//build URL
	uri, err := r.URL(client, r.Options.Values)
	if err != nil {
		return nil, err
	}

	//build request
	req, err := http.NewRequest(r.Method, uri, r.Body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", provider.UserAgent.Join())
	for key, values := range r.Options.Headers {
		req.Header[key] = values
	}
	for key, value := range provider.AuthenticatedHeaders() {
		req.Header.Set(key, value)
	}

	resp, err := provider.HTTPClient.Do(req)
	if err != nil {
		return nil, err
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
