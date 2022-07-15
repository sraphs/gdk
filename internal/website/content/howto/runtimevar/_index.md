---
title: "Runtime Configuration"
date: 2019-07-11T12:00:00-07:00
lastmod: 2020-12-23T12:00:00-07:00
showInSidenav: true
toc: true
---

The [`runtimevar` package][] provides an easy and portable way to watch runtime
configuration variables. This guide shows how to work with runtime configuration
variables using the Go CDK.

<!--more-->

Subpackages contain driver implementations of runtimevar for various services,
including Cloud and on-prem solutions. You can develop your application locally
using [`filevar`][] or [`constantvar`][], then deploy it to multiple Cloud
providers with minimal initialization reconfiguration.

[`runtimevar` package]: https://godoc.org/github.com/sraphs/gdk/runtimevar
[`filevar`]: https://godoc.org/github.com/sraphs/gdk/runtimevar/filevar
[`constantvar`]: https://godoc.org/github.com/sraphs/gdk/runtimevar/constantvar

## Opening a Variable {#opening}

The first step in watching a variable is to instantiate a portable
[`*runtimevar.Variable`][] for your service.

The easiest way to do so is to use [`runtimevar.OpenVariable`][] and a service-specific URL pointing
to the variable, making sure you ["blank import"][] the driver package to link
it in.

```go
import (
	"github.com/sraphs/gdk/runtimevar"
	_ "github.com/sraphs/gdk/runtimevar/<driver>"
)
...
v, err := runtimevar.OpenVariable(context.Background(), "<driver-url>")
if err != nil {
    return fmt.Errorf("could not open variable: %v", err)
}
defer v.Close()
// v is a *runtimevar.Variable; see usage below
...
```

See [Concepts: URLs][] for general background and the [guide below][]
for URL usage for each supported service.

Alternatively, if you need fine-grained control
over the connection settings, you can call the constructor function in the
driver package directly (like `etcdvar.OpenVariable`).

```go
import "github.com/sraphs/gdk/runtimevar/<driver>"
...
v, err := <driver>.OpenVariable(...)
...
```

You may find the [`wire` package][] useful for managing your initialization code
when switching between different backing services.

See the [guide below][] for constructor usage for each supported service.

When opening the variable, you can provide a [decoder][] parameter (either as a
[query parameter][] for URLs, or explicitly to the constructor) to specify
whether the raw value stored in the variable is interpreted as a `string`, a
`[]byte`, or as JSON. Here's an example of using a JSON encoder:

{{< goexample src="github.com/sraphs/gdk/runtimevar.Example_jsonDecoder" imports="0" >}}

[`*runtimevar.Variable`]: https://godoc.org/github.com/sraphs/gdk/runtimevar#Variable
[`runtimevar.OpenVariable`]: https://godoc.org/github.com/sraphs/gdk/runtimevar#OpenVariable
["blank import"]: https://golang.org/doc/effective_go.html#blank_import
[Concepts: URLs]: {{< ref "/concepts/urls.md" >}}
[decoder]: https://godoc.org/github.com/sraphs/gdk/runtimevar#Decoder
[guide below]: {{< ref "#services" >}}
[query parameter]: https://godoc.org/github.com/sraphs/gdk/runtimevar#DecoderByName
[`wire` package]: http://github.com/google/wire

## Using a Variable {#using}

Once you have opened a `runtimevar.Variable` for the provider you want, you can
use it portably.

### Latest {#latest}

The easiest way to a `Variable` is to use the [`Variable.Latest`][] method. It
returns the latest good [`Snapshot`][] of the variable value, blocking if no
good value has *ever* been detected. The dynamic type of `Snapshot.Value`
depends on the decoder you provided when creating the `Variable`.

{{< goexample src="github.com/sraphs/gdk/runtimevar.ExampleVariable_Latest" imports="0" >}}

To avoid blocking, you can pass an already-`Done` context. You can also use
[`Variable.CheckHealth`][], which reports as healthy when `Latest` will
return a value without blocking.

[`Variable.Latest`]: https://godoc.org/github.com/sraphs/gdk/runtimevar#Variable.Latest
[`Variable.CheckHealth`]: https://godoc.org/github.com/sraphs/gdk/runtimevar#Variable.CheckHealth
[`Snapshot`]: https://godoc.org/github.com/sraphs/gdk/runtimevar#Snapshot

### Watch {#watch}

`Variable` also has a [`Watch`][] method for obtaining the value of a variable;
it has different semantics than `Latest` and may be useful in some scenarios. We
recommend starting with `Latest` as it's conceptually simpler to work with.

[`Watch`]: https://godoc.org/github.com/sraphs/gdk/runtimevar#Variable.Watch

## Other Usage Samples

* [CLI Sample](https://github.com/sraphs/gdk/tree/master/samples/gocdk-runtimevar)
* [Guestbook sample](https://github.com/sraphs/gdk/tutorials/guestbook/)
* [runtimevar package examples](https://godoc.org/github.com/sraphs/gdk/runtimevar#pkg-examples)

## Supported Services {#services}

### etcd {#etcd}

*NOTE*: Support for `etcd` has been temporarily dropped due to dependency
issues. See https://github.com/sraphs/gdk/issues/2914.

You can use `runtimevar.etcd` in Go CDK version `v0.20.0` or earlier.

### HTTP {#http}

`httpvar` supports watching a variable via an HTTP request. Use
`runtimevar.OpenVariable` with a regular URL starting with `http` or `https`.
`httpvar` will periodically make an HTTP `GET` request to that URL, with the
`decode` URL parameter removed (if present).

{{< goexample "github.com/sraphs/gdk/runtimevar/httpvar.Example_openVariableFromURL" >}}

#### HTTP Constructor {#http-ctor}

The [`httpvar.OpenVariable`][] constructor opens a variable with a `http.Client`
and a URL.

{{< goexample "github.com/sraphs/gdk/runtimevar/httpvar.ExampleOpenVariable" >}}

[`httpvar.OpenVariable`]: https://godoc.org/github.com/sraphs/gdk/runtimevar/httpvar#OpenVariable

### Blob {#blob}

`blobvar` supports watching a variable based on the contents of a
[Go CDK blob][]. Set the environment variable `BLOBVAR_BUCKET_URL` to the URL
of the bucket, and then use `runtimevar.OpenVariable` as shown below.
`blobvar` will periodically re-fetch the contents of the blob.

{{< goexample "github.com/sraphs/gdk/runtimevar/blobvar.Example_openVariableFromURL" >}}

[Go CDK blob]: https://github.com/sraphs/gdk/howto/blob/

You can also use [`blobvar.OpenVariable`][].

[`blobvar.OpenVariable`]: https://godoc.org/github.com/sraphs/gdk/runtimevar/blobvar#OpenVariable

### Local {#local}

You can create an in-memory variable (useful for testing) using `constantvar`:

{{< goexample "github.com/sraphs/gdk/runtimevar/constantvar.Example_openVariableFromURL" >}}

Alternatively, you can create a variable based on the contents of a file using
`filevar`:

{{< goexample "github.com/sraphs/gdk/runtimevar/filevar.Example_openVariableFromURL" >}}
