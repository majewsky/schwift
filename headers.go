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
	"reflect"

	"github.com/majewsky/schwift/headers"
)

//AccountHeaders contains the headers for an account. The Headers attribute
//contains the actual set of headers that was returned from a HEAD or GET
//request on the account, and will be sent by a PUT or POST request. The other
//attributes allow type-safe access to well-known headers, as noted in the tags
//next to each field.
//
//Follow the link on the Headers attribute for the documentation of the Get(),
//Set(), Del(), Clear() methods on this type.
type AccountHeaders struct {
	headers.Headers
	BytesUsed      headers.Uint64Readonly `schwift:"X-Account-Bytes-Used"`
	ContainerCount headers.Uint64Readonly `schwift:"X-Account-Container-Count"`
	Metadata       headers.Metadata       `schwift:"X-Account-Meta-"`
	ObjectCount    headers.Uint64Readonly `schwift:"X-Account-Object-Count"`
	QuotaBytes     headers.Uint64         `schwift:"X-Account-Meta-Quota-Bytes"`
	TempURLKey     headers.String         `schwift:"X-Account-Meta-Temp-URL-Key"`
	TempURLKey2    headers.String         `schwift:"X-Account-Meta-Temp-URL-Key-2"`
	//forbid initialization as struct literal (must use NewAccountHeaders)
	initialized bool
}

//NewAccountHeaders prepares a new AccountHeaders instance.
//
//WARNING: Always use this function to construct AccountHeaders instances.
//Failure to do so will result in uncontrolled crashes!
func NewAccountHeaders() AccountHeaders {
	var ah AccountHeaders
	ah.Headers = make(headers.Headers)
	initializeByReflection(&ah)
	ah.initialized = true
	return ah
}

//Validate returns headers.MalformedHeaderError if the value of any well-known
//header does not conform to its data type. This is called automatically by
//Schwift when preparing an AccountHeaders instance from a GET/HEAD response,
//so you usually do not need to do it yourself. You will get the validation error
//from the Account method doing the request, e.g. Headers() or List().
func (ah AccountHeaders) Validate() error {
	return validateByReflection(&ah)
}

//ContainerHeaders contains the headers for a container. The Headers attribute
//contains the actual set of headers that was returned from a HEAD or GET
//request on the container, and will be sent by a PUT or POST request. The
//other attributes allow type-safe access to well-known headers, as noted in
//the tags next to each field.
//
//Follow the link on the Headers attribute for the documentation of the Get(),
//Set(), Del(), Clear() methods on this type.
type ContainerHeaders struct {
	headers.Headers
	Metadata headers.Metadata `schwift:"X-Container-Meta-"`
	//TODO map well-known headers
	//forbid initialization as struct literal (must use NewContainerHeaders)
	initialized bool
}

//NewContainerHeaders prepares a new ContainerHeaders instance.
//
//WARNING: Always use this function to construct ContainerHeaders instances.
//Failure to do so will result in uncontrolled crashes!
func NewContainerHeaders() ContainerHeaders {
	var ch ContainerHeaders
	ch.Headers = make(headers.Headers)
	initializeByReflection(&ch)
	ch.initialized = true
	return ch
}

//Validate returns headers.MalformedHeaderError if the value of any well-known
//header does not conform to its data type. This is called automatically by
//Schwift when preparing an ContainerHeaders instance from a GET/HEAD response,
//so you usually do not need to do it yourself. You will get the validation error
//from the Container method doing the request, e.g. Headers() or List().
func (ch ContainerHeaders) Validate() error {
	return validateByReflection(&ch)
}

type fieldInfo struct {
	FieldName  string
	HeaderName string
}

func initializeByReflection(value interface{}) {
	rv := reflect.ValueOf(value).Elem()
	hdrs := rv.FieldByName("Headers").Interface().(headers.Headers)

	foreachTaggedField(value, func(fieldPtr interface{}, info fieldInfo) error {
		base := reflect.ValueOf(fieldPtr).Elem().FieldByName("Base").Addr().Interface().(*headers.Base)
		base.H = hdrs
		base.K = info.HeaderName
		return nil
	})
}

type validator interface {
	Validate() error
}

func validateByReflection(value interface{}) error {
	return foreachTaggedField(value, func(fieldPtr interface{}, info fieldInfo) error {
		if validator, ok := fieldPtr.(validator); ok {
			err := validator.Validate()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureInitializedByReflection(value interface{}) {
	initialized := reflect.ValueOf(value).FieldByName("initialized").Bool()
	if !initialized {
		msg := "values of type %T MUST be initialized with the corresponding New...() function"
		panic(fmt.Sprintf(msg, value, value))
	}
}

func foreachTaggedField(value interface{}, callback func(fieldPtr interface{}, info fieldInfo) error) error {
	rv := reflect.ValueOf(value).Elem()

	//iterate over the struct fields
	for idx := 0; idx < rv.NumField(); idx++ {
		fieldType := rv.Type().Field(idx)
		headerName := fieldType.Tag.Get("schwift")

		if headerName != "" {
			fieldPtr := rv.Field(idx).Addr().Interface()
			err := callback(fieldPtr, fieldInfo{
				FieldName:  fieldType.Name,
				HeaderName: headerName,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
