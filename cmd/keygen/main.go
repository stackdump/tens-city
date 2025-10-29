package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/stackdump/tens-city/internal/ethsig"
)

func main() {
	keystorePath := flag.String("out", "", "output path for keystore file")
	passphrase := flag.String("pass", "", "passphrase for encrypting the keystore")
	privkeyHex := flag.String("privkey", "", "optional: hex-encoded private key to import (generates new if not provided)")
	flag.Parse()

	if *keystorePath == "" {
		fmt.Fprintln(os.Stderr, "Error: -out is required")
		flag.Usage()
		os.Exit(1)
	}

	if *passphrase == "" {
		fmt.Fprintln(os.Stderr, "Error: -pass is required")
		flag.Usage()
		os.Exit(1)
	}

	if *privkeyHex != "" {
		// Import existing private key
		privKey, err := ethsig.LoadPrivateKeyFromHex(*privkeyHex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load private key: %v\n", err)
			os.Exit(1)
		}
		addr := ethsig.GetAddressFromPrivateKey(privKey)
		fmt.Printf("Importing address: %s\n", addr)

		if err := ethsig.CreateKeystore(privKey, *passphrase, *keystorePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create keystore: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Generate a new key
		privKey, err := ethsig.GenerateNewKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate key: %v\n", err)
			os.Exit(1)
		}

		addr := ethsig.GetAddressFromPrivateKey(privKey)
		fmt.Printf("Generated address: %s\n", addr)

		if err := ethsig.CreateKeystore(privKey, *passphrase, *keystorePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create keystore: %v\n", err)
			os.Exit(1)
		}

		// Print the private key for backup/testing
		hexKey := ethsig.PrivateKeyToHex(privKey)
		fmt.Printf("Private key (backup): 0x%s\n", hexKey)
	}

	fmt.Printf("Keystore created: %s\n", *keystorePath)
}
