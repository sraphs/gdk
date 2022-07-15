package hashivault_test

import (
	"context"
	"log"

	"github.com/hashicorp/vault/api"

	"github.com/sraphs/gdk/secrets"
	"github.com/sraphs/gdk/secrets/hashivault"
)

func ExampleOpenKeeper() {
	// PRAGMA: This example is used on gocloud.dev; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On gocloud.dev, hide lines until the next blank line.
	ctx := context.Background()

	// Get a client to use with the Vault API.
	client, err := hashivault.Dial(ctx, &hashivault.Config{
		Token: "CLIENT_TOKEN",
		APIConfig: api.Config{
			Address: "http://127.0.0.1:8200",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Construct a *secrets.Keeper.
	keeper := hashivault.OpenKeeper(client, "my-key", nil)
	defer keeper.Close()
}

func Example_openFromURL() {
	// PRAGMA: This example is used on gocloud.dev; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On gocloud.dev, add a blank import: _ "gocloud.dev/secrets/hashivault"
	// PRAGMA: On gocloud.dev, hide lines until the next blank line.
	ctx := context.Background()

	keeper, err := secrets.OpenKeeper(ctx, "hashivault://mykey")
	if err != nil {
		log.Fatal(err)
	}
	defer keeper.Close()
}
