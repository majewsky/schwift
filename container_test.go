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
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestContainerExistence(t *testing.T) {
	testWithAccount(t, func(a *Account) {
		c := a.Container(getRandomName())

		exists, err := c.Exists()
		expectError(t, err, "")
		expectBool(t, exists, false)

		_, err = c.Headers()
		expectError(t, err, "expected 204 response, got 404 instead")
		expectBool(t, Is(err, http.StatusNotFound), true)
		expectBool(t, Is(err, http.StatusNoContent), false)

		err = c.Create(NewContainerHeaders(), nil)
		expectError(t, err, "")

		exists, err = c.Exists()
		expectError(t, err, "")
		expectBool(t, exists, true)
	})
}

func getRandomName() string {
	var buf [16]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		panic(err.Error())
	}
	return hex.EncodeToString(buf[:])
}
