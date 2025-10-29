# Ethereum Signing Support for Sealed Objects

This document describes how to use Ethereum wallet support and cryptographic signing with the sealing pipeline.

## Overview

The `seal` CLI has been extended to support signing sealed JSON-LD objects using Ethereum-style signatures. This provides cryptographic proof of authorship and integrity for sealed objects.

## Features

- **Keystore Support**: Load encrypted Ethereum keystores with passphrase protection
- **Private Key Support**: Direct private key import (for testing/development)
- **Signature Storage**: Signatures are automatically stored alongside sealed objects
- **Multiple Signing Modes**: Support for both personal_sign (EIP-191) and raw signing
- **Key Generation**: Utility to create new keystores or import existing keys

## Usage

### Creating a Keystore

Use the `keygen` utility to create a new encrypted keystore:

```bash
# Generate a new key
go build ./cmd/keygen
./keygen -out keystore.json -pass "your-secure-passphrase"

# Import an existing private key
./keygen -out keystore.json -pass "your-secure-passphrase" -privkey 0x1234...
```

The keystore file is encrypted using the industry-standard scrypt algorithm with secure parameters.

### Signing with Keystore

To seal and sign an object using a keystore:

```bash
go build ./cmd/seal
./seal -in examples/petrinet.jsonld \
       -store data \
       -user alice \
       -gist demo \
       -keystore keystore.json
```

You will be prompted for your passphrase securely (no echo to terminal).

### Signing with Private Key

For testing or automation (less secure):

```bash
./seal -in examples/petrinet.jsonld \
       -store data \
       -privkey 0x1234567890abcdef...
```

### Signing Modes

By default, signatures use the `personal_sign` method (EIP-191), which is compatible with most Ethereum wallets:

```bash
./seal -in input.jsonld -keystore keystore.json -personal-sign=true
```

For raw signatures (keccak256 of canonical bytes):

```bash
./seal -in input.jsonld -keystore keystore.json -personal-sign=false
```

## Signature Storage

Signatures are stored in JSON format at:
```
{store}/o/signatures/{cid}.json
```

Example signature metadata:
```json
{
  "signature": "0x7f2276541eca66e8797e19cb914e2d4842e24a9911df0113951cd1d8d4cc2674...",
  "signerAddress": "0x650137860C9424a8F4Da8a6bB2cF1cCC73077F2D",
  "usePersonalSign": true
}
```

## Verification

To verify a signature (using the ethsig package in your code):

```go
import "github.com/stackdump/tens-city/internal/ethsig"
import "github.com/stackdump/tens-city/internal/store"

// Read the object and signature
st := store.NewFSStore("data")
canonical, _ := st.ReadCanonical(cid)
sigMeta, _ := st.ReadSignature(cid)

// Verify
valid, err := ethsig.VerifyEthereumSignature(
    canonical,
    sigMeta.Signature,
    sigMeta.SignerAddress,
    sigMeta.UsePersonalSign,
)
```

## CLI Flags Reference

### seal command

- `-keystore <path>`: Path to encrypted Ethereum keystore file
- `-privkey <hex>`: Hex-encoded private key (0x-prefix optional)
- `-personal-sign`: Use personal_sign (EIP-191) mode (default: true)
- `-store <dir>`: Base directory for storing objects and signatures
- `-in <file>`: Input JSON-LD file (use `-` for stdin)
- `-user <login>`: Optional user login for container updates
- `-gist <slug>`: Optional gist slug for container updates
- `-pretty`: Write human-readable canonical N-Quads file

### keygen command

- `-out <path>`: Output path for keystore file (required)
- `-pass <passphrase>`: Passphrase for encrypting the keystore (required)
- `-privkey <hex>`: Import existing private key instead of generating new one

## Security Considerations

### Keystore Files

- Keystore files are encrypted with scrypt using standard Ethereum parameters
- Files are created with restrictive permissions (0600 - owner read/write only)
- Never commit keystore files to version control
- Store keystore files securely and back them up

### Passphrases

- Use strong, unique passphrases for keystore encryption
- Passphrases are read securely without terminal echo
- For automation, consider using environment variables or secure vaults

### Private Keys

- The `-privkey` flag should only be used for testing/development
- Never expose private keys in command history, logs, or scripts
- For production, always use keystores with passphrase protection

### Signing Modes

- **personal_sign (EIP-191)**: Compatible with wallet software like MetaMask
  - Adds prefix: `"\x19Ethereum Signed Message:\n{len(message)}"`
  - Recommended for user-facing applications
  
- **Raw signing**: Direct keccak256 hash of canonical bytes
  - Use when you control both signing and verification
  - Slightly more efficient but less wallet-compatible

## Integration Example

```bash
# 1. Create a keystore
./keygen -out alice-keystore.json -pass "secure-password"

# 2. Seal and sign an object
./seal -in examples/petrinet.jsonld \
       -store data \
       -user alice \
       -gist petri-demo \
       -keystore alice-keystore.json \
       -pretty

# 3. Verify the signature was stored
cat data/o/signatures/{cid}.json
```

## Testing

Run the test suite:

```bash
go test ./internal/ethsig/...
go test ./internal/store/...
go test ./...
```

## Examples

See the `examples/` directory for sample JSON-LD files to seal and sign.
