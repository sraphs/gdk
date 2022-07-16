package constantvar_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/constantvar"
)

func ExampleNew() {
	// Construct a *runtimevar.Variable that always returns "hello world".
	v := constantvar.New("hello world")
	defer v.Close()

	// We can now read the current value of the variable from v.
	snapshot, err := v.Latest(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(snapshot.Value.(string))

	// Output:
	// hello world
}

func ExampleNewBytes() {
	// Construct a *runtimevar.Variable with a []byte.
	v := constantvar.NewBytes([]byte(`hello world`), runtimevar.BytesDecoder)
	defer v.Close()

	// We can now read the current value of the variable from v.
	snapshot, err := v.Latest(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("byte slice of length %d\n", len(snapshot.Value.([]byte)))

	// Output:
	// byte slice of length 11
}

func ExampleNewError() {
	// Construct a runtimevar.Variable that always returns errFake.
	var errFake = errors.New("my error")
	v := constantvar.NewError(errFake)
	defer v.Close()

	// We can now use Watch to read the current value of the variable
	// from v. Note that Latest would block here since it waits for
	// a "good" value, and v will never get one.
	_, err := v.Watch(context.Background())
	if err == nil {
		log.Fatal("Expected an error!")
	}
	fmt.Println(err)

	// Output:
	// runtimevar (code=Unknown): my error
}

func Example_openVariableFromURL() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, add a blank import: _ "github.com/sraphs/gdk/runtimevar/constantvar"
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// runtimevar.OpenVariable creates a *runtimevar.Variable from a URL.

	v, err := runtimevar.OpenVariable(ctx, "constant://?val=hello+world&decoder=string")
	if err != nil {
		log.Fatal(err)
	}
	defer v.Close()
	// PRAGMA: On github.com/sraphs/gdk, hide the rest of the function.
	snapshot, err := v.Latest(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(snapshot.Value.(string))

	// Output
	// hello world
}
