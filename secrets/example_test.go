package secrets_test

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/status"

	"github.com/sraphs/gdk/secrets"
	"github.com/sraphs/gdk/secrets/localsecrets"
)

func Example() {
	ctx := context.Background()

	// Construct a *secrets.Keeper from one of the secrets subpackages.
	// This example uses localsecrets.
	sk, err := localsecrets.NewRandomKey()
	if err != nil {
		log.Fatal(err)
	}
	keeper := localsecrets.NewKeeper(sk)
	defer keeper.Close()

	// Now we can use keeper to Encrypt.
	plaintext := []byte("Go CDK Secrets")
	ciphertext, err := keeper.Encrypt(ctx, plaintext)
	if err != nil {
		log.Fatal(err)
	}

	// And/or Decrypt.
	decrypted, err := keeper.Decrypt(ctx, ciphertext)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(decrypted))

	// Output:
	// Go CDK Secrets
}

func Example_errorAs() {
	// This example is specific to the gcpkms implementation; it
	// demonstrates access to the underlying google.golang.org/grpc/status.Status
	// type.
	// The types exposed for As by gcpkms are documented in
	// https://godoc.org/github.com/sraphs/gdk/secrets/gcpkms#hdr-As
	ctx := context.Background()

	const url = "gcpkms://projects/proj/locations/global/keyRings/test/ring/wrongkey"
	keeper, err := secrets.OpenKeeper(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer keeper.Close()

	plaintext := []byte("Go CDK secrets")
	_, err = keeper.Encrypt(ctx, plaintext)
	if err != nil {
		var s *status.Status
		if keeper.ErrorAs(err, &s) {
			fmt.Println(s.Code())
		}
	}
}

func ExampleKeeper_Encrypt() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()
	var keeper *secrets.Keeper

	plainText := []byte("Secrets secrets...")
	cipherText, err := keeper.Encrypt(ctx, plainText)
	if err != nil {
		log.Fatal(err)
	}

	// PRAGMA: On github.com/sraphs/gdk, hide the rest of the function.
	_ = cipherText
}

func ExampleKeeper_Decrypt() {
	// PRAGMA: This example is used on github.com/sraphs/gdk; PRAGMA comments adjust how it is shown and can be ignored.
	// PRAGMA: On github.com/sraphs/gdk, hide lines until the next blank line.
	ctx := context.Background()
	var keeper *secrets.Keeper

	var cipherText []byte // obtained from elsewhere and random-looking
	plainText, err := keeper.Decrypt(ctx, cipherText)
	if err != nil {
		log.Fatal(err)
	}

	// PRAGMA: On github.com/sraphs/gdk, hide the rest of the function.
	_ = plainText
}
