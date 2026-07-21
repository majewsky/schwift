// SPDX-FileCopyrightText: 2018 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"go.xyrillian.de/schwift/v2"
)

func TestObjectLifecycle(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		objectName := getRandomName()
		o := c.Object(objectName)

		expectString(t, o.Name(), objectName)
		expectString(t, o.FullName(), c.Name()+"/"+objectName)
		if o.Container() != c {
			t.Errorf("expected o.Container() = %#v, got %#v instead\n", c, o.Container())
		}
		expectObjectExistence(t, o, false)

		_, err := o.Headers(t.Context())
		expectError(t, err, fmt.Sprintf("could not HEAD %q in Swift: expected 200 response, got 404 instead", o.FullName()))
		expectBool(t, schwift.Is(err, http.StatusNotFound), true)
		expectBool(t, schwift.Is(err, http.StatusNoContent), false)

		// DELETE should be idempotent and not return success on non-existence, but
		// OpenStack LOVES to be inconsistent with everything (including, notably, itself)
		err = o.Delete(t.Context(), nil, nil)
		expectError(t, err, fmt.Sprintf("could not DELETE %q in Swift: expected 204 response, got 404 instead: <html><h1>Not Found</h1><p>The resource could not be found.</p></html>", o.FullName()))

		err = o.Upload(t.Context(), bytes.NewReader([]byte("test")), nil, nil)
		expectSuccess(t, err)

		expectObjectExistence(t, o, true)

		err = o.Delete(t.Context(), nil, nil)
		expectSuccess(t, err)
	})
}

func TestObjectUpload(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		// test upload with bytes.Reader
		obj := c.Object("upload1")
		err := obj.Upload(t.Context(), bytes.NewReader(objectExampleContent), nil, nil)
		expectSuccess(t, err)
		expectObjectContent(t, obj, objectExampleContent)

		// test upload with bytes.Buffer
		obj = c.Object("upload2")
		err = obj.Upload(t.Context(), bytes.NewBuffer(objectExampleContent), nil, nil)
		expectSuccess(t, err)
		expectObjectContent(t, obj, objectExampleContent)

		// test upload with strings.Reader
		obj = c.Object("upload3")
		err = obj.Upload(t.Context(), strings.NewReader(string(objectExampleContent)), nil, nil)
		expectSuccess(t, err)
		expectObjectContent(t, obj, objectExampleContent)

		// test upload with opaque io.Reader
		obj = c.Object("upload4")
		err = obj.Upload(t.Context(), opaqueReader{bytes.NewReader(objectExampleContent)}, nil, nil)
		expectSuccess(t, err)
		expectObjectContent(t, obj, objectExampleContent)

		// test upload with io.Writer
		obj = c.Object("upload5")
		err = obj.UploadFromWriter(t.Context(), nil, nil, func(w io.Writer) error {
			_, err := w.Write(objectExampleContent)
			return err
		})
		expectSuccess(t, err)
		expectObjectContent(t, obj, objectExampleContent)

		// test upload with empty reader (should create zero-byte-sized object)
		obj = c.Object("upload6")
		err = obj.Upload(t.Context(), eofReader{}, nil, nil)
		expectSuccess(t, err)
		expectObjectContent(t, obj, nil)

		// test upload without reader (should create zero-byte-sized object)
		obj = c.Object("upload7")
		err = obj.Upload(t.Context(), nil, nil, nil)
		expectSuccess(t, err)
		expectObjectContent(t, obj, nil)
	})
}

type eofReader struct{}

func (r eofReader) Read([]byte) (int, error) {
	return 0, io.EOF
}

type opaqueReader struct {
	b *bytes.Reader
}

func (r opaqueReader) Read(buf []byte) (int, error) {
	return r.b.Read(buf)
}

func TestObjectDownload(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		// upload example object
		obj := c.Object("example")
		err := obj.Upload(t.Context(), bytes.NewReader(objectExampleContent), nil, nil)
		expectSuccess(t, err)

		// test download as string
		str, err := obj.Download(t.Context(), nil).AsString()
		expectSuccess(t, err)
		expectString(t, str, string(objectExampleContent))

		// test download as byte slice
		buf, err := obj.Download(t.Context(), nil).AsByteSlice()
		expectSuccess(t, err)
		expectString(t, string(buf), string(objectExampleContent))

		// test download as io.ReadCloser slice
		reader, err := obj.Download(t.Context(), nil).AsReadCloser()
		expectSuccess(t, err)
		buf = make([]byte, 4)
		_, err = reader.Read(buf)
		expectSuccess(t, err)
		expectString(t, string(buf), string(objectExampleContent[0:4]))
		_, err = reader.Read(buf)
		expectSuccess(t, err)
		expectString(t, string(buf), string(objectExampleContent[4:8]))
		buf, err = io.ReadAll(reader)
		expectSuccess(t, err)
		expectString(t, string(buf), string(objectExampleContent[8:]))
	})
}

func TestObjectUpdate(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		obj := c.Object("example")

		// test that metadata update fails for non-existing object
		newHeaders := schwift.NewObjectHeaders()
		newHeaders.ContentType().Set("application/json")
		err := obj.Update(t.Context(), newHeaders, nil)
		expectBool(t, schwift.Is(err, http.StatusNotFound), true)
		expectError(t, err, fmt.Sprintf("could not POST %q in Swift: expected 202 response, got 404 instead: <html><h1>Not Found</h1><p>The resource could not be found.</p></html>", obj.FullName()))

		// create object
		err = obj.Upload(t.Context(), nil, nil, nil)
		expectSuccess(t, err)

		hdr, err := obj.Headers(t.Context())
		expectSuccess(t, err)
		expectString(t, hdr.ContentType().Get(), "application/octet-stream")

		// now the metadata update should work
		err = obj.Update(t.Context(), newHeaders, nil)
		expectSuccess(t, err)
		obj.Invalidate()
		hdr, err = obj.Headers(t.Context())
		expectSuccess(t, err)
		expectString(t, hdr.ContentType().Get(), "application/json")
	})
}

func TestObjectCopy(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		obj1 := c.Object("location1")
		err := obj1.Upload(t.Context(), bytes.NewReader(objectExampleContent), nil, nil)
		expectSuccess(t, err)
		expectObjectExistence(t, obj1, true)

		obj2 := c.Object("location2")
		expectSuccess(t, obj1.CopyTo(t.Context(), obj2, nil, nil))
		expectObjectExistence(t, obj1, true)
		expectObjectExistence(t, obj2, true)
		expectObjectContent(t, obj2, objectExampleContent)
	})
}

func TestSymlinkOperations(t *testing.T) {
	testWithContainer(t, func(c *schwift.Container) {
		// create a test object that we can link to
		obj1 := c.Object("target")
		err := obj1.Upload(t.Context(), bytes.NewReader(objectExampleContent), nil, nil)
		expectSuccess(t, err)
		expectObjectExistence(t, obj1, true)

		// create a symlink
		obj2 := c.Object("symlink")
		expectSuccess(t, obj2.SymlinkTo(t.Context(), obj1, nil, nil))
		expectObjectExistence(t, obj2, true)
		expectObjectSymlink(t, obj2, obj1)
		expectObjectContent(t, obj2, objectExampleContent)

		// overwrite symlink with normal object
		otherContent := []byte("abc")
		expectSuccess(t, obj2.Upload(t.Context(), bytes.NewReader(otherContent), nil, nil))
		expectObjectExistence(t, obj2, true)
		expectObjectSymlink(t, obj2, nil)
		expectObjectContent(t, obj2, otherContent)

		// overwrite normal object with symlink
		expectSuccess(t, obj2.SymlinkTo(t.Context(), obj1, nil, nil))
		expectObjectExistence(t, obj2, true)
		expectObjectSymlink(t, obj2, obj1)
		expectObjectContent(t, obj2, objectExampleContent)

		// deep-copy symlink
		obj3 := c.Object("copy")
		expectSuccess(t, obj2.CopyTo(t.Context(), obj3, nil, nil))
		expectObjectExistence(t, obj3, true)
		expectObjectSymlink(t, obj3, nil)
		expectObjectContent(t, obj3, objectExampleContent)

		// shallow-copy symlink
		expectSuccess(t, obj2.CopyTo(t.Context(), obj3, &schwift.CopyOptions{
			ShallowCopySymlinks: true,
		}, nil))
		expectObjectExistence(t, obj3, true)
		expectObjectSymlink(t, obj3, obj1)
		expectObjectContent(t, obj3, objectExampleContent)

		// delete symlink
		expectSuccess(t, obj2.Delete(t.Context(), nil, nil))
		expectObjectExistence(t, obj2, false)
	})
}

////////////////////////////////////////////////////////////////////////////////
// helpers

func expectObjectExistence(t *testing.T, obj *schwift.Object, expectedExists bool) {
	t.Helper()
	obj.Invalidate()
	actualExists, err := obj.Exists(t.Context())
	expectSuccess(t, err)
	expectBool(t, actualExists, expectedExists)
}

func expectObjectContent(t *testing.T, obj *schwift.Object, expected []byte) {
	t.Helper()
	str, err := obj.Download(t.Context(), nil).AsString()
	expectSuccess(t, err)
	expectString(t, str, string(expected))
	obj.Invalidate()
	hdr, err := obj.Headers(t.Context())
	expectSuccess(t, err)
	if !hdr.IsLargeObject() {
		expectString(t, hdr.Etag().Get(), etagOf(expected))
	}
}

func expectObjectSymlink(t *testing.T, source, expectedTarget *schwift.Object) {
	t.Helper()
	_, target, err := source.SymlinkHeaders(t.Context())
	if expectedTarget == nil {
		switch err {
		case nil:
			if target != nil {
				t.Errorf("expected %s to not be a symlink, but found symlink to %s\n",
					source.FullName(), target.FullName())
			}
		default:
			t.Errorf("got unexpected error from Object.SymlinkHeaders() for %s: %s\n",
				source.FullName(), err.Error())
		}
	} else {
		if err != nil {
			t.Errorf("expected %s to be a symlink to %s, but Object.SymlinkHeaders() returned error: %s\n",
				source.FullName(), expectedTarget.FullName(), err.Error())
		} else if target.FullName() != expectedTarget.FullName() {
			t.Errorf("expected %s to be a symlink to %s, but got target %s\n",
				source.FullName(), expectedTarget.FullName(), target.FullName())
		}
	}
}
