# Schwift

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
