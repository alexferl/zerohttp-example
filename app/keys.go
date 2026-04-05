package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	zl "github.com/alexferl/zerohttp/log"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

func loadOrCreateKey(path string) (jwk.Set, error) {
	if _, err := os.Stat(path); err == nil {
		zl.GetGlobalLogger().Info("Loading existing JWT key", zl.F("path", path))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file: %w", err)
		}

		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block")
		}

		privateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse EC private key: %w", err)
		}

		key, err := jwk.Import(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to import private key: %w", err)
		}

		keySet := jwk.NewSet()
		if err := keySet.AddKey(key); err != nil {
			return nil, fmt.Errorf("failed to add key to set: %w", err)
		}
		return keySet, nil
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create key file: %w", err)
	}

	if err := pem.Encode(file, pemBlock); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close key file: %w", err)
	}
	zl.GetGlobalLogger().Info("Created new JWT key", zl.F("path", path))

	key, err := jwk.Import(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to import private key: %w", err)
	}

	keySet := jwk.NewSet()
	if err := keySet.AddKey(key); err != nil {
		return nil, fmt.Errorf("failed to add key to set: %w", err)
	}
	return keySet, nil
}
