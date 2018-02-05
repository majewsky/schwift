# Schwift

[![GoDoc](https://godoc.org/github.com/majewsky/schwift?status.svg)](https://godoc.org/github.com/majewsky/schwift)

this is a Go client library for [OpenStack Swift](https://github.com/openstack/swift). I made this after growing
frustrated with the bad API design of [`ncw/swift`](https://github.com/ncw/swift).

<p style="color:red;font-weight:bold">WARNING: This is in a pre-alpha stage and neither complete nor tested.</p>

## Installation

You can get this with `go get github.com/majewsky/schwift`. When using this in an application, vendoring is recommended.

## Usage

This library uses [Gophercloud](https://github.com/gophercloud/gophercloud) to handle authentication, so to use Schwift, you have to first build a `gophercloud.ServiceClient` and then pass that to `schwift.Account()` to get a handle on the Swift account.

For example, to connect to Swift using OpenStack Keystone authentication:

```go
import (
  "log"

  "github.com/gophercloud/gophercloud"
  "github.com/gophercloud/gophercloud/openstack"
  "github.com/majewsky/schwift"
)

authOptions, err := openstack.AuthOptionsFromEnv()
handle(err)
provider, err := openstack.AuthenticatedClient(authOptions)
handle(err)
client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts {})
handle(err)

account, err := schwift.AccountFromClient(client)
handle(err)
```

To connect to Swift using Swift's built-in authentication:

```go
import (
  "log"

  "github.com/gophercloud/gophercloud"
  "github.com/gophercloud/gophercloud/openstack"
  "github.com/gophercloud/gophercloud/openstack/objectstore/v1/swauth"
  "github.com/majewsky/schwift"
)

provider, err := openstack.NewClient("http://swift.example.com:8080")
handle(err)
client, err := swauth.NewObjectStorageV1(provider, swauth.AuthOpts {
    User: "project:user",
    Key:  "password",
})
handle(err)

account, err := schwift.AccountFromClient(client)
handle(err)
```

From this point, follow the [API documentation](https://godoc.org/github.com/majewsky/schwift) for what you can do with
the `schwift.Account` object.

## Why another Swift client library?

The most popular Swift client library is [`ncw/swift`](https://github.com/ncw/swift). I have [used
it](https://github.com/docker/distribution/pull/2441) [extensively](https://github.com/sapcc/swift-http-import) and my
main gripe with it is that its API is designed around single tasks (like "get content of body as string") which are each
modeled as single functions. Since you cannot add arguments to an existing function without breaking backwards
compatibility, this means that if the existing functions do not cover your usecase, you have to add another function to
do basically the same thing. When you're trying to do something that's not one of the 10 most common things, you're
going to run into dead ends where the API does not allow you do specify that one URL parameter that you need. Like that
one day [when I filed five issues in a row because every function in the API that I tried turned out to be missing
something](https://github.com/ncw/swift/issues?utf8=%E2%9C%93&q=is%3Aissue+author%3Amajewsky+created%3A2017-11).

This library uses Gophercloud for authentication (which solves one problem that ncw/swift has, namely that you cannot
use the Keystone token that ncw/swift fetches for talking to other OpenStack services), but besides the auth code, it
avoids pretty much all other parts of Gophercloud, because it too has fatal design flaws:

- The API is modeled around individual requests and responses, which means that there will probably never be support for
  advanced features like large objects unless you're willing to do all the footwork yourself.
- The built-in error handling paves over any useful error messages that the server might return. For example, when you
  get a 404 response, `err.Error()` only says [`Resource not
  found`](https://github.com/gophercloud/gophercloud/blob/4a3f5ae58624b68283375060dad06a214b05a32b/errors.go#L112). To
  get the actual server error message, you have to use `err.(*gophercloud.ErrUnexpectedResponseCode).Body` which is
  absolutely obvious.
- The implementation is quite unidiomatic. It all looks like a Java developer's first Go project. For example, to resume
  the error handling example, [all of
  this](https://github.com/gophercloud/gophercloud/blob/4a3f5ae58624b68283375060dad06a214b05a32b/errors.go#L65-L178)
  should be deleted without replacement because `ErrUnexpectedResponseCode` does the same without paving over the server
  error message. Most other types in that module should probably be deleted as well (there is no plausible reason for
  requiring all error types to inherit from a `BaseError`; after all, this is Go, not Java).
