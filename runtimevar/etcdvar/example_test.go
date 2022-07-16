package etcdvar_test

import (
	"context"
	"log"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/etcdvar"
)

func ExampleOpenVariable() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.

	// Connect to the etcd server.
	client, err := clientv3.NewFromURL("http://your.etcd.server:9999")
	if err != nil {
		log.Fatal(err)
	}

	// Construct a *runtimevar.Variable that watches the variable.
	v, err := etcdvar.OpenVariable(client, "cfg-variable-name", runtimevar.StringDecoder, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
}

func Example_openVariableFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/runtimevar/etcdvar"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// runtimevar.OpenVariable creates a *runtimevar.Variable from a URL.
	// The default opener connects to an etcd server based on the environment
	// variable ETCD_SERVER_URL.

	v, err := runtimevar.OpenVariable(ctx, "etcd://myvarname?decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
}
