package localsecrets_test

import (
	"context"
	"log"

	"github.com/sraphs/gdk/secrets"
	"github.com/sraphs/gdk/secrets/localsecrets"
)

func ExampleNewKeeper() {
	// PRAGMA: This example is used on gocloud.dev; PRAGMA comments adjust how it is shown and can be ignored.

	secretKey, err := localsecrets.NewRandomKey()
	if err != nil {
		log.Fatal(err)
	}
	keeper := localsecrets.NewKeeper(secretKey)
	defer keeper.Close()
}

func Example_openFromURL() {
	// PRAGMA: This example is used on gocloud.dev; PRAGMA comments adjust how it is shown and can be ignored.

	// PRAGMA: On gocloud.dev, add a blank import: _ "gocloud.dev/secrets/localsecrets"

	// PRAGMA: On gocloud.dev, hide lines until the next blank line.
	ctx := context.Background()

	// Using "base64key://", a new random key will be generated.
	randomKeyKeeper, err := secrets.OpenKeeper(ctx, "base64key://")
	if err != nil {
		log.Fatal(err)
	}
	defer randomKeyKeeper.Close()

	// Otherwise, the URL hostname must be a base64-encoded key, of length 32 bytes when decoded.
	// Note that base64.URLEncode should be used, to avoid URL-unsafe characters.
	savedKeyKeeper, err := secrets.OpenKeeper(ctx, "base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=")
	if err != nil {
		log.Fatal(err)
	}
	defer savedKeyKeeper.Close()
}
