// Package activitypub provides ActivityPub protocol support for tens-city.
// It enables federation with Mastodon and other ActivityPub-compatible platforms.
package activitypub

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds ActivityPub configuration for an actor
type Config struct {
	// Actor identity
	Username    string // e.g., "myork"
	Domain      string // e.g., "blog.stackdump.com"
	DisplayName string // e.g., "stackdump"
	Summary     string // Actor bio/description

	// URLs
	ProfileURL string // HTML profile page URL
	IconURL    string // Avatar/icon URL

	// Key storage
	KeyPath string // Path to store/load RSA private key

	// Software info (for NodeInfo)
	SoftwareName    string
	SoftwareVersion string
}

// Actor represents an ActivityPub actor
type Actor struct {
	config     *Config
	privateKey *rsa.PrivateKey
}

// NewActor creates a new ActivityPub actor with the given configuration.
// It loads or generates an RSA keypair for HTTP signatures.
func NewActor(config *Config) (*Actor, error) {
	actor := &Actor{config: config}

	// Load or generate keypair
	if err := actor.loadOrGenerateKey(); err != nil {
		return nil, fmt.Errorf("failed to initialize keys: %w", err)
	}

	return actor, nil
}

// loadOrGenerateKey loads an existing private key or generates a new one
func (a *Actor) loadOrGenerateKey() error {
	if a.config.KeyPath == "" {
		return fmt.Errorf("key path not configured")
	}

	// Try to load existing key
	keyData, err := os.ReadFile(a.config.KeyPath)
	if err == nil {
		// Parse existing key
		block, _ := pem.Decode(keyData)
		if block == nil {
			return fmt.Errorf("failed to decode PEM block")
		}

		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			// Try PKCS8 format
			keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return fmt.Errorf("failed to parse private key: %w", err)
			}
			var ok bool
			key, ok = keyInterface.(*rsa.PrivateKey)
			if !ok {
				return fmt.Errorf("key is not RSA")
			}
		}

		a.privateKey = key
		return nil
	}

	// Generate new key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}
	a.privateKey = key

	// Save key to file
	if err := a.saveKey(); err != nil {
		return fmt.Errorf("failed to save key: %w", err)
	}

	return nil
}

// saveKey saves the private key to the configured path
func (a *Actor) saveKey() error {
	// Ensure directory exists
	dir := filepath.Dir(a.config.KeyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Encode key to PEM
	keyBytes := x509.MarshalPKCS1PrivateKey(a.privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	// Write to file with restrictive permissions
	return os.WriteFile(a.config.KeyPath, pem.EncodeToMemory(block), 0600)
}

// PublicKeyPEM returns the public key in PEM format
func (a *Actor) PublicKeyPEM() string {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&a.privateKey.PublicKey)
	if err != nil {
		return ""
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	return string(pem.EncodeToMemory(block))
}

// ID returns the actor's ActivityPub ID (URI)
func (a *Actor) ID() string {
	return fmt.Sprintf("https://%s/users/%s", a.config.Domain, a.config.Username)
}

// KeyID returns the actor's public key ID
func (a *Actor) KeyID() string {
	return a.ID() + "#main-key"
}

// InboxURL returns the actor's inbox URL
func (a *Actor) InboxURL() string {
	return a.ID() + "/inbox"
}

// OutboxURL returns the actor's outbox URL
func (a *Actor) OutboxURL() string {
	return a.ID() + "/outbox"
}

// FollowersURL returns the actor's followers collection URL
func (a *Actor) FollowersURL() string {
	return a.ID() + "/followers"
}

// FollowingURL returns the actor's following collection URL
func (a *Actor) FollowingURL() string {
	return a.ID() + "/following"
}

// Config returns the actor's configuration
func (a *Actor) Config() *Config {
	return a.config
}
