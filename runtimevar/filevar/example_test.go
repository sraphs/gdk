package filevar_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/filevar"
)

func ExampleOpenVariable() {
	// Create a temporary file to hold our config.
	f, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte("hello world")); err != nil {
		log.Fatal(err)
	}

	// Construct a *runtimevar.Variable pointing at f.
	v, err := filevar.OpenVariable(f.Name(), runtimevar.StringDecoder, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()

	// We can now read the current value of the variable from v.
	snapshot, err := v.Latest(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	// runtimevar.Snapshot.Value is decoded to a string.
	fmt.Println(snapshot.Value.(string))

	// Output:
	// hello world
}

func Example_openVariableFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/runtimevar/filevar"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// runtimevar.OpenVariable creates a *runtimevar.Variable from a URL.

	v, err := runtimevar.OpenVariable(ctx, "file:///path/to/config.txt?decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
}
