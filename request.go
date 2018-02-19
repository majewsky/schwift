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
)

//RequestOptions contains additional headers and values for a request.
type RequestOptions struct {
	Values url.Values
}

func cloneRequestOptions(orig *RequestOptions) *RequestOptions {
	result := RequestOptions{
		Values: make(url.Values),
	}
	if orig != nil {
		for k, v := range orig.Values {
			result.Values[k] = v
		}
	}
	return &result
}

//Request contains the parameters that can be set in a request to the Swift API.
type Request struct {
	Method        string //"GET", "HEAD", "PUT", "POST" or "DELETE"
	ContainerName string //empty for requests on accounts
	ObjectName    string //empty for requests on accounts/containers
	Headers       http.Header
	Options       *RequestOptions
	Body          io.Reader
	//ExpectStatusCodes can be left empty to disable this check, otherwise
	//schwift.UnexpectedStatusCodeError may be returned.
	ExpectStatusCodes []int
	//DrainResponseBody can be set if the caller is not interested in the
	//response body. This is implied for Response.StatusCode == 204.
	DrainResponseBody bool
}

//URL returns the full URL for this request.
func (r Request) URL(client Client, values url.Values) (string, error) {
	uri, err := url.Parse(client.EndpointURL())
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

//Do executes this request on the given Client.
func (r Request) Do(client Client) (*http.Response, error) {
	//build URL
	var values url.Values
	if r.Options != nil {
		values = r.Options.Values
	}
	uri, err := r.URL(client, values)
	if err != nil {
		return nil, err
	}

	//build request
	req, err := http.NewRequest(r.Method, uri, r.Body)
	if err != nil {
		return nil, err
	}

	for k, v := range r.Headers {
		req.Header[k] = v
	}
	if r.Body != nil {
		req.Header.Set("Expect", "100-continue")
	}

	resp, err := client.Do(req)
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
			var err error
			if r.DrainResponseBody || resp.StatusCode == 204 {
				err = drainResponseBody(resp)
			}
			return resp, err
		}
	}

	//unexpected status code -> generate error
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
