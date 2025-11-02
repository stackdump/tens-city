package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/stackdump/tens-city/internal/ethsig"
	"github.com/stackdump/tens-city/internal/seal"
	"github.com/stackdump/tens-city/internal/store"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	// CLI flags
	inPath := flag.String("in", "-", "input file (JSON-LD). Use - for stdin")
	outDir := flag.String("store", "data", "base directory to store sealed objects and containers")
	user := flag.String("user", "", "optional user login (for updating /u/{login}/g/{slug}/latest)")
	gist := flag.String("gist", "", "optional gist slug (used with --user to update latest and history)")
	pretty := flag.Bool("pretty", false, "write canonicalized normalized n-quads to a human-readable file alongside the object")

	// Ethereum signing flags
	keystore := flag.String("keystore", "", "path to Ethereum keystore file (encrypted JSON)")
	privkey := flag.String("privkey", "", "hex-encoded private key (less secure, for testing only)")
	usePersonalSign := flag.Bool("personal-sign", true, "use personal_sign (EIP-191) for signing (default true)")

	flag.Parse()

	var data []byte
	var err error
	if *inPath == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("reading stdin: %v", err)
		}
	} else {
		data, err = os.ReadFile(*inPath)
		if err != nil {
			log.Fatalf("reading file %s: %v", *inPath, err)
		}
	}

	// initialize filesystem store
	st := store.NewFSStore(*outDir)

	// run sealing pipeline: canonicalize (URDNA2015), compute CIDv1 (base32) and return canonical bytes
	cidStr, canonicalBytes, err := seal.SealJSONLD(data)
	if err != nil {
		log.Fatalf("seal failed: %v", err)
	}

	// Sign the canonical bytes if a key is provided
	var signature string
	var signerAddr string
	if *keystore != "" || *privkey != "" {
		var sigErr error
		signature, signerAddr, sigErr = signCanonicalBytes(canonicalBytes, *keystore, *privkey, *usePersonalSign)
		if sigErr != nil {
			log.Fatalf("signing failed: %v", sigErr)
		}
		fmt.Printf("signed by: %s\n", signerAddr)
		fmt.Printf("signature: %s\n", signature)
	}

	// persist raw JSON-LD and canonical form
	if err := st.SaveObject(cidStr, data, canonicalBytes); err != nil {
		log.Fatalf("store SaveObject: %v", err)
	}

	// If we have a signature, save it alongside the object
	if signature != "" {
		if err := st.SaveSignature(cidStr, signature, signerAddr, *usePersonalSign); err != nil {
			log.Fatalf("store SaveSignature: %v", err)
		}
		fmt.Printf("signature stored at: %s\n", st.SignaturePath(cidStr))
	}

	// optionally update user/gist pointer and history
	if *user != "" && *gist != "" {
		if err := st.UpdateLatest(*user, *gist, cidStr); err != nil {
			log.Fatalf("UpdateLatest failed: %v", err)
		}
		if err := st.AppendHistory(*user, *gist, cidStr); err != nil {
			log.Fatalf("AppendHistory failed: %v", err)
		}
		fmt.Printf("sealed as %s and updated /u/%s/g/%s/latest\n", cidStr, *user, *gist)
	} else {
		fmt.Printf("sealed as %s\n", cidStr)
	}

	// optionally write the canonical form to a human readable file next to the canonical payload
	if *pretty {
		hf := fmt.Sprintf("%s/canonical/%s.nq", *outDir, cidStr)
		if err := os.MkdirAll(fmt.Sprintf("%s/canonical", *outDir), 0o755); err != nil {
			log.Fatalf("mkdir canonical: %v", err)
		}
		if err := os.WriteFile(hf, canonicalBytes, 0o644); err != nil {
			log.Fatalf("write canonical file: %v", err)
		}
		fmt.Printf("wrote canonical n-quads to %s\n", hf)
	}

	// echo the saved object path
	objPath := st.ObjectPath(cidStr)
	fmt.Printf("object stored at: %s\n", objPath)

	// show first 512 bytes of canonical form on stdout for convenience
	fmt.Println("---- canonical (first 512 bytes) ----")
	r := io.LimitReader(io.NopCloser(io.NewSectionReader(os.NewFile(0, ""), 0, 1)), 0)
	_ = r // silence linter for unused; canonical printed below instead
	if len(canonicalBytes) > 0 {
		if len(canonicalBytes) > 512 {
			fmt.Printf("%s\n", canonicalBytes[:512])
		} else {
			fmt.Printf("%s\n", canonicalBytes)
		}
	}
}

// signCanonicalBytes signs the canonical bytes using the provided keystore or private key.
// Returns (signature, signerAddress, error).
func signCanonicalBytes(canonicalBytes []byte, keystorePath, privkeyHex string, usePersonalSign bool) (string, string, error) {
	var privkeyHexStr string
	var signerAddr string

	if keystorePath != "" {
		// Load address from keystore to show the user
		addr, err := ethsig.GetAddressFromKeystore(keystorePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to read keystore address: %w", err)
		}
		fmt.Printf("Using keystore: %s\n", keystorePath)
		fmt.Printf("Address: %s\n", addr)

		// Prompt for passphrase
		passphrase, err := readPassphrase("Enter passphrase: ")
		if err != nil {
			return "", "", fmt.Errorf("failed to read passphrase: %w", err)
		}

		// Load the private key
		privkey, err := ethsig.LoadPrivateKeyFromKeystore(keystorePath, passphrase)
		if err != nil {
			return "", "", fmt.Errorf("failed to load keystore: %w", err)
		}

		privkeyHexStr = ethsig.PrivateKeyToHex(privkey)
		signerAddr = ethsig.GetAddressFromPrivateKey(privkey)
	} else if privkeyHex != "" {
		// Load from hex-encoded private key
		privkey, err := ethsig.LoadPrivateKeyFromHex(privkeyHex)
		if err != nil {
			return "", "", fmt.Errorf("failed to load private key from hex: %w", err)
		}
		privkeyHexStr = ethsig.PrivateKeyToHex(privkey)
		signerAddr = ethsig.GetAddressFromPrivateKey(privkey)
	} else {
		return "", "", fmt.Errorf("either -keystore or -privkey must be specified for signing")
	}

	// Sign the canonical bytes
	var signature string
	var err error
	if usePersonalSign {
		signature, err = ethsig.SignPersonalWithPrivKey(canonicalBytes, privkeyHexStr)
	} else {
		signature, err = ethsig.SignCanonicalWithPrivKey(canonicalBytes, privkeyHexStr)
	}
	if err != nil {
		return "", "", fmt.Errorf("signing failed: %w", err)
	}

	return signature, signerAddr, nil
}

// readPassphrase reads a passphrase from stdin without echoing.
// Falls back to regular input if terminal is not available.
func readPassphrase(prompt string) (string, error) {
	fmt.Print(prompt)

	// Try to read from terminal without echo
	if terminal.IsTerminal(int(syscall.Stdin)) {
		passBytes, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println() // Print newline after password input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(passBytes)), nil
	}

	// Fallback to regular input for non-terminal (e.g., piped input)
	reader := bufio.NewReader(os.Stdin)
	pass, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pass), nil
}
