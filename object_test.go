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
	"bytes"
	"net/http"
	"testing"
)

func TestObjectLifecycle(t *testing.T) {
	testWithContainer(t, func(c *Container) {
		objectName := getRandomName()
		o := c.Object(objectName)

		expectString(t, o.Name(), objectName)
		if o.Container() != c {
			t.Errorf("expected o.Container() = %#v, got %#v instead\n", c, o.Container())
		}

		exists, err := o.Exists()
		expectError(t, err, "")
		expectBool(t, exists, false)

		_, err = o.Headers()
		expectError(t, err, "expected 204 response, got 404 instead")
		expectBool(t, Is(err, http.StatusNotFound), true)
		expectBool(t, Is(err, http.StatusNoContent), false)

		//DELETE should be idempotent and not return success on non-existence, but
		//OpenStack LOVES to be inconsistent with everything (including, notably, itself)
		err = o.Delete(nil, nil)
		expectError(t, err, "expected 204 response, got 404 instead: <html><h1>Not Found</h1><p>The resource could not be found.</p></html>")

		err = o.Upload(bytes.NewReader([]byte("test")), nil, nil)
		expectError(t, err, "")

		exists, err = o.Exists()
		expectError(t, err, "")
		expectBool(t, exists, true)

		err = o.Delete(nil, nil)
		expectError(t, err, "")
	})
}
