package app

import (
	"os"
	"path/filepath"
	"testing"

	zl "github.com/alexferl/zerohttp/log"
	"github.com/alexferl/zerohttp/zhtest"
)

func TestLoadOrCreateKey_GeneratesNewKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	zl.SetGlobalLogger(zl.NewDefaultLogger())

	keySet, err := loadOrCreateKey(keyPath)
	zhtest.AssertNoError(t, err)
	zhtest.AssertNotNil(t, keySet)
	zhtest.AssertEqual(t, 1, keySet.Len())

	info, err := os.Stat(keyPath)
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, os.FileMode(0o600), info.Mode().Perm())

	key, ok := keySet.Key(0)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertNotNil(t, key)
}

func TestLoadOrCreateKey_LoadsExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	zl.SetGlobalLogger(zl.NewDefaultLogger())

	keySet1, err := loadOrCreateKey(keyPath)
	zhtest.AssertNoError(t, err)

	keySet2, err := loadOrCreateKey(keyPath)
	zhtest.AssertNoError(t, err)
	zhtest.AssertNotNil(t, keySet2)

	key1, ok1 := keySet1.Key(0)
	key2, ok2 := keySet2.Key(0)
	zhtest.AssertTrue(t, ok1)
	zhtest.AssertTrue(t, ok2)
	zhtest.AssertNotNil(t, key1)
	zhtest.AssertNotNil(t, key2)
}

func TestLoadOrCreateKey_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	err := os.WriteFile(keyPath, []byte("invalid pem data"), 0o600)
	zhtest.AssertNoError(t, err)

	_, err = loadOrCreateKey(keyPath)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "failed to decode PEM block")
}

func TestLoadOrCreateKey_InvalidECKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	pemData := `-----BEGIN EC PRIVATE KEY-----
ICAgIGludmFsaWQga2V5IGRhdGE=
-----END EC PRIVATE KEY-----`
	err := os.WriteFile(keyPath, []byte(pemData), 0o600)
	zhtest.AssertNoError(t, err)

	_, err = loadOrCreateKey(keyPath)
	zhtest.AssertError(t, err)
	zhtest.AssertErrorContains(t, err, "failed to parse EC private key")
}

func TestLoadOrCreateKey_ReadOnlyDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "subdir", "test.key")

	err := os.MkdirAll(filepath.Dir(keyPath), 0o500)
	zhtest.AssertNoError(t, err)

	_, err = loadOrCreateKey(keyPath)
	zhtest.AssertError(t, err)
}
