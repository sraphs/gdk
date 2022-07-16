package httpvar_test

import (
	"context"
	"log"
	"net/http"

	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/httpvar"
)

func ExampleOpenVariable() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.

	// Create an HTTP.Client
	httpClient := http.DefaultClient

	// Construct a *runtimevar.Variable that watches the page.
	v, err := httpvar.OpenVariable(httpClient, "http://example.com", runtimevar.StringDecoder, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
}

func Example_openVariableFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/runtimevar/httpvar"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// runtimevar.OpenVariable creates a *runtimevar.Variable from a URL.
	// The default opener connects to an etcd server based on the environment
	// variable ETCD_SERVER_URL.

	v, err := runtimevar.OpenVariable(ctx, "http://myserver.com/foo.txt?decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
}
