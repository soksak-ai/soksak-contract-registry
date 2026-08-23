package registry

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func signedFixture(t *testing.T) (Registry, Trust) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	value := registryFixture()
	value.Signature.KeyID = "test-key"
	if err := Sign(&value, private); err != nil {
		t.Fatal(err)
	}
	return value, Trust{RegistryID: "official", KeyID: "test-key", PublicKey: public}
}
func TestRegistryAuthenticationAndContinuity(t *testing.T) {
	value, trust := signedFixture(t)
	receipt, err := Verify(value, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	value.Sequence = 2
	if _, err := Verify(value, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), nil); err == nil {
		t.Fatal("mutation accepted")
	}
	value, _ = signedFixture(t)
	if _, err := Verify(value, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), &HighWater{Sequence: 2, Digest: receipt.Digest}); err == nil {
		t.Fatal("rollback accepted")
	}
}
