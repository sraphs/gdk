package runtimevar_test

import (
	"context"
	"fmt"
	"log"

	"github.com/sraphs/gdk/runtimevar"
	_ "github.com/sraphs/gdk/runtimevar/constantvar"
)

func Example_openVariableFromURL() {
	// Connect to a Variable using a URL.
	// This example uses "constantvar", an in-memory implementation.
	// We need to add a blank import line to register the constantvar driver's
	// URLOpener, which implements runtimevar.VariableURLOpener:
	// import _ "github.com/sraphs/gdk/runtimevar/constantvar"
	// constantvar registers for the "constant" scheme.
	// All runtimevar.OpenVariable URLs also work with "runtimevar+" or "runtimevar+variable+" prefixes,
	// e.g., "runtimevar+constant://..." or "runtimevar+variable+constant://...".
	ctx := context.Background()
	v, err := runtimevar.OpenVariable(ctx, "constant://?val=hello+world&decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	// Now we can use the Variable as normal.
	snapshot, err := v.Latest(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// It's safe to cast the Value to string since we used the string decoder.
	fmt.Printf("%s\n", snapshot.Value.(string))

	// Output:
	// hello world
}
