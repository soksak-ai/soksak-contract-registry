package registry

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func signedFixture(t *testing.T) (SignedRegistry, Trust) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	payload := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	document := SignedRegistry{Registry: payload, IssuedAt: "2026-08-21T00:00:00Z", ExpiresAt: "2026-09-21T00:00:00Z", KeyID: "test-key", Algorithm: "ed25519"}
	if err := Sign(&document, private); err != nil {
		t.Fatal(err)
	}
	return document, Trust{RegistryID: "official", KeyID: "test-key", PublicKey: public}
}

func TestSignedRegistryCoversReleases(t *testing.T) {
	document, trust := signedFixture(t)
	result, err := Verify(document, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	document.Registry.Sequence = 2
	if _, err := Verify(document, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), nil); err == nil {
		t.Fatal("registry mutation kept a valid signature")
	}
	if result.Sequence != 1 || result.Digest == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestRegistryTrustRejectsRollbackAndEquivocation(t *testing.T) {
	document, trust := signedFixture(t)
	result, err := Verify(document, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(document, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), &HighWater{Sequence: 2, Digest: result.Digest}); err == nil {
		t.Fatal("rollback was accepted")
	}
	if _, err := Verify(document, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), &HighWater{Sequence: 1, Digest: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}); err == nil {
		t.Fatal("equivocation was accepted")
	}
}

func TestRegistryTrustRejectsExpiredAndWrongIdentity(t *testing.T) {
	document, trust := signedFixture(t)
	if _, err := Verify(document, trust, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), nil); err == nil {
		t.Fatal("expired registry was accepted")
	}
	trust.RegistryID = "other"
	if _, err := Verify(document, trust, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), nil); err == nil {
		t.Fatal("wrong registry identity was accepted")
	}
}
