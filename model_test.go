package registry

import (
	"encoding/base64"
	"testing"
)

func reference(id string) ReleaseReference {
	return ReleaseReference{ID: id, Version: "0.0.1", URL: "https://github.com/example/" + id + "/releases/download/v0.0.1/release.json", Size: 1, SHA256: "a" + string(make([]byte, 0)) + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
}
func registryFixture() Registry {
	return Registry{ID: "official", Sequence: 1, IssuedAt: "2026-08-21T00:00:00Z", ExpiresAt: "2026-09-21T00:00:00Z", Plugins: []Plugin{}, Signature: Signature{Algorithm: "ed25519", KeyID: "test-key", Value: base64.StdEncoding.EncodeToString(make([]byte, 64))}}
}
func TestRegistryContainsPluginsAndDirectRuntimeDependenciesOnly(t *testing.T) {
	value := registryFixture()
	value.Plugins = []Plugin{{ReleaseReference: reference("weather-plugin"), RuntimeDependencies: &RuntimeDependencies{Sidecars: []ReleaseReference{reference("weather-sidecar")}}}}
	if err := Validate(value); err != nil {
		t.Fatal(err)
	}
}
func TestRegistryRejectsHistoryAndVersionLocators(t *testing.T) {
	value := registryFixture()
	value.Plugins = []Plugin{{ReleaseReference: reference("weather-plugin")}, {ReleaseReference: reference("weather-plugin")}}
	if Validate(value) == nil {
		t.Fatal("duplicate plugin accepted")
	}
	invalid := reference("weather-plugin")
	invalid.Version = "latest"
	value.Plugins = []Plugin{{ReleaseReference: invalid}}
	if Validate(value) == nil {
		t.Fatal("latest accepted")
	}
}
