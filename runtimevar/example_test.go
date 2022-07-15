package runtimevar_test

import (
	"context"
	"fmt"
	"log"

	"github.com/sraphs/gdk/runtimevar"
	"github.com/sraphs/gdk/runtimevar/constantvar"
	"github.com/sraphs/gdk/secrets"

	runtimeconfig "google.golang.org/genproto/googleapis/cloud/runtimeconfig/v1beta1"
	"google.golang.org/grpc/status"

	_ "github.com/sraphs/gdk/runtimevar/filevar"
)

func Example_jsonDecoder() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()

	// Config is the sample config struct we're going to parse our JSON into.
	type Config struct {
		Host string
		Port int
	}

	// A sample JSON config that will decode into Config.
	const jsonConfig = `{"Host": "github.com/sraphs/gdk", "Port": 8080}`

	// Construct a Decoder that decodes raw bytes into our config.
	decoder := runtimevar.NewDecoder(Config{}, runtimevar.JSONDecode)

	// Next, a construct a *Variable using a constructor or URL opener.
	// This example uses constantvar.
	// If you're using a URL opener, you can't decode JSON into a struct, but
	// you can use the query parameter "decoder=jsonmap" to decode into a map.
	v := constantvar.NewBytes([]byte(jsonConfig), decoder)
	defer v.Close()
	// snapshot.Value will be of type Config.

	// PRAGMA: On github.com/sraphs/gdk, hide the rest of the function.
	snapshot, err := v.Latest(ctx)
	if err != nil {
		log.Fatalf("Error in retrieving variable: %v", err)
	}
	fmt.Printf("Config: %+v\n", snapshot.Value.(Config))

	// Output:
	// Config: {Host:github.com/sraphs/gdk Port:8080}
}

func Example_stringDecoder() {
	// Construct a *Variable using a constructor from one of the
	// runtimevar subpackages. This example uses constantvar.
	// The variable value is of type string, so we use StringDecoder.
	v := constantvar.NewBytes([]byte("hello world"), runtimevar.StringDecoder)
	defer v.Close()

	// Call Latest to retrieve the value.
	snapshot, err := v.Latest(context.Background())
	if err != nil {
		log.Fatalf("Error in retrieving variable: %v", err)
	}
	// snapshot.Value will be of type string.
	fmt.Printf("%q\n", snapshot.Value.(string))

	// Output:
	// "hello world"
}

func ExampleVariable_Latest() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	var v *runtimevar.Variable

	snapshot, err := v.Latest(context.Background())
	if err != nil {
		log.Fatalf("Error in retrieving variable: %v", err)
	}
	// PRAGMA: On github.com/sraphs/gdk, hide the rest of the function.
	_ = snapshot
}

func ExampleSnapshot_As() {
	// This example is specific to the file implementation; it
	// demonstrates access to the underlying
	// google.golang.org/genproto/googleapis/cloud/runtimeconfig.Variable type.
	// The types exposed for As by file are documented in
	// https://godoc.org/github.com/sraphs/gdk/runtimevar/file#hdr-As
	ctx := context.Background()

	const url = "file://proj/config/key"
	v, err := runtimevar.OpenVariable(ctx, url)
	if err != nil {
		log.Fatal(err)
	}

	s, err := v.Latest(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var rcv *runtimeconfig.Variable
	if s.As(&rcv) {
		fmt.Println(rcv.UpdateTime)
	}
}

func ExampleVariable_ErrorAs() {
	// This example is specific to the file implementation; it
	// demonstrates access to the underlying google.golang.org/grpc/status.Status
	// type.
	// The types exposed for As by file are documented in
	// https://godoc.org/github.com/sraphs/gdk/runtimevar/file#hdr-As
	ctx := context.Background()

	const url = "file://proj/wrongconfig/key"
	v, err := runtimevar.OpenVariable(ctx, url)
	if err != nil {
		log.Fatal(err)
	}

	_, err = v.Watch(ctx)
	if err != nil {
		var s *status.Status
		if v.ErrorAs(err, &s) {
			fmt.Println(s.Code())
		}
	}
}

func ExampleVariable_Watch() {
	// Construct a *Variable using a constructor from one of the
	// runtimevar subpackages. This example uses constantvar.
	// The variable value is of type string, so we use StringDecoder.
	v := constantvar.NewBytes([]byte("hello world"), runtimevar.StringDecoder)
	defer v.Close()

	// Call Watch in a loop from a background goroutine to see all changes,
	// including errors.
	//
	// You can use this for logging, or to trigger behaviors when the
	// config changes.
	//
	// Note that Latest always returns the latest "good" config, so seeing
	// an error from Watch doesn't mean that Latest will return one.
	go func() {
		for {
			snapshot, err := v.Watch(context.Background())
			if err == runtimevar.ErrClosed {
				// v has been closed; exit.
				return
			}
			if err == nil {
				// Casting to a string here because we used StringDecoder.
				log.Printf("New config: %v", snapshot.Value.(string))
			} else {
				log.Printf("Error loading config: %v", err)
				// Even though there's been an error loading the config,
				// v.Latest will continue to return the latest "good" value.
			}
		}
	}()
}

func ExampleDecryptDecode() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	var keeper *secrets.Keeper

	decodeFunc := runtimevar.DecryptDecode(keeper, runtimevar.StringDecode)
	decoder := runtimevar.NewDecoder("", decodeFunc)

	// PRAGMA: On github.com/sraphs/gdk, hide the rest of the function.
	_ = decoder
}
