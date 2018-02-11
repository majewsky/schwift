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
	"net/http"
)

//Object represents a Swift object.
type Object struct {
	c    *Container
	name string
	//cache
	headers *ObjectHeaders
}

//Object returns a handle to the object with the given name within this
//container. This function does not issue any HTTP requests, and therefore cannot
//ensure that the object exists. Use the Exists() function to check for the
//container's existence.
func (c *Container) Object(name string) *Object {
	return &Object{c: c, name: name}
}

//Container returns a handle to the container this object is stored in.
func (o *Object) Container() *Container {
	return o.c
}

//Name returns the object name. This does not parse the name in any way; if you
//want only the basename portion of the object name, use package path in
//conjunction with this function. For example:
//
//	obj := account.Container("docs").Object("2018-02-10/invoice.pdf")
//	obj.Name()            //returns "2018-02-10/invoice.pdf"
//	path.Base(obj.Name()) //returns            "invoice.pdf"
func (o *Object) Name() string {
	return o.name
}

//FullName returns the container name and object name joined together with a
//slash. This identifier is used by Swift in several places (DLO manifests,
//symlink targets, etc.) to refer to an object within an account. For example:
//
//	obj := account.Container("docs").Object("2018-02-10/invoice.pdf")
//	obj.Name()     //returns      "2018-02-10/invoice.pdf"
//	obj.FullName() //returns "docs/2018-02-10/invoice.pdf"
func (o *Object) FullName() string {
	return o.c.name + "/" + o.name
}

//Exists checks if this object exists, potentially by issuing a HEAD request
//if no Headers() have been cached yet.
func (o *Object) Exists() (bool, error) {
	_, err := o.Headers()
	if Is(err, http.StatusNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

//Headers returns the ObjectHeaders for this object. If the ObjectHeaders
//has not been cached yet, a HEAD request is issued on the object.
//
//This operation fails with http.StatusNotFound if the object does not exist.
func (o *Object) Headers() (ObjectHeaders, error) {
	if o.headers != nil {
		return *o.headers, nil
	}

	resp, err := Request{
		Method:            "HEAD",
		ContainerName:     o.c.name,
		ObjectName:        o.name,
		ExpectStatusCodes: []int{204},
	}.Do(o.c.a.client)
	if err != nil {
		return ObjectHeaders{}, err
	}

	headers := ObjectHeaders(headersFromHTTP(resp.Header))
	err = headers.Validate()
	if err != nil {
		return headers, err
	}
	o.headers = &headers
	return *o.headers, nil
}

//Update updates the object's headers using a POST request. To add URL
//parameters, pass a non-nil *RequestOptions.
//
//If you are not sure whether the container exists, use Create() instead.
//
//A successful POST request implies Invalidate() since it may change metadata.
func (o *Object) Update(headers ObjectHeaders, opts *RequestOptions) error {
	_, err := Request{
		Method:            "POST",
		ContainerName:     o.c.name,
		ObjectName:        o.name,
		Headers:           headersToHTTP(headers),
		Options:           opts,
		ExpectStatusCodes: []int{204},
	}.Do(o.c.a.client)
	if err == nil {
		o.Invalidate()
	}
	return err
}

//Upload creates the object using a PUT request. To add URL parameters, pass
//a non-nil *RequestOptions.
//
//This function can be used regardless of whether the object exists or not.
//
//A successful PUT request implies Invalidate() since it may change metadata.
func (o *Object) Upload(content io.Reader, headers ObjectHeaders, opts *RequestOptions) error {
	//TODO check hash
	_, err := Request{
		Method:            "PUT",
		ContainerName:     o.c.name,
		ObjectName:        o.name,
		Headers:           headersToHTTP(headers),
		Options:           opts,
		Body:              content,
		ExpectStatusCodes: []int{201},
		DrainResponseBody: true,
	}.Do(o.c.a.client)
	if err == nil {
		o.Invalidate()
	}
	return err
}

//Delete deletes the object using a DELETE request. To add URL parameters,
//pass a non-nil *RequestOptions.
//
//This operation fails with http.StatusNotFound if the object does not exist.
//
//A successful DELETE request implies Invalidate().
func (o *Object) Delete(headers ObjectHeaders, opts *RequestOptions) error {
	_, err := Request{
		Method:            "DELETE",
		ContainerName:     o.c.name,
		ObjectName:        o.name,
		Headers:           headersToHTTP(headers),
		Options:           opts,
		ExpectStatusCodes: []int{204},
	}.Do(o.c.a.client)
	if err == nil {
		o.c.Invalidate()
	}
	return err
}

//Invalidate clears the internal cache of this Object instance. The next call
//to Headers() on this instance will issue a HEAD request on the object.
func (o *Object) Invalidate() {
	o.headers = nil
}

//TODO Object.Copy(), Object.Move(), Object.Download()
//TODO does Object.Upload() have the right API?
