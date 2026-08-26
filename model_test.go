package registry

import (
	"encoding/base64"
	"strings"
	"testing"
)

func reference(id string) ReleaseReference {
	return ReleaseReference{ID: id, Version: "0.0.1", Size: 1, SHA256: strings.Repeat("a", 64)}
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
func registryBody(reference string) string {
	return `{"id":"official","sequence":1,"issuedAt":"2026-08-21T00:00:00Z","expiresAt":"2026-09-21T00:00:00Z","plugins":[` + reference + `],"signature":{"algorithm":"ed25519","keyId":"test-key","value":"` + base64.StdEncoding.EncodeToString(make([]byte, 64)) + `"}}`
}
func TestParseRejectsLocationInReleaseReference(t *testing.T) {
	digest := strings.Repeat("a", 64)
	located := `{"id":"weather-plugin","version":"0.0.1","url":"https://github.com/soksak-ai/weather-plugin/releases/download/v0.0.1/release.json","size":1,"sha256":"` + digest + `"}`
	_, err := Parse([]byte(registryBody(located)))
	if err == nil || !strings.Contains(err.Error(), `unknown field "url"`) {
		t.Fatalf("url accepted: %v", err)
	}
	unlocated := `{"id":"weather-plugin","version":"0.0.1","size":1,"sha256":"` + digest + `"}`
	if _, err := Parse([]byte(registryBody(unlocated))); err != nil {
		t.Fatal(err)
	}
}
func TestReferencesRemainSortedAndUnique(t *testing.T) {
	value := registryFixture()
	plugin := Plugin{ReleaseReference: reference("weather-plugin"), RuntimeDependencies: &RuntimeDependencies{Sidecars: []ReleaseReference{reference("weather-sidecar-b"), reference("weather-sidecar-a")}}}
	value.Plugins = []Plugin{plugin}
	if Validate(value) == nil {
		t.Fatal("unsorted sidecars accepted")
	}
	plugin.RuntimeDependencies.Sidecars = []ReleaseReference{reference("weather-sidecar-a"), reference("weather-sidecar-a")}
	if Validate(value) == nil {
		t.Fatal("duplicate sidecars accepted")
	}
	plugin.RuntimeDependencies.Sidecars = []ReleaseReference{reference("weather-sidecar-a"), reference("weather-sidecar-b")}
	if err := Validate(value); err != nil {
		t.Fatal(err)
	}
	value.Plugins = []Plugin{{ReleaseReference: reference("weather-plugin-b")}, {ReleaseReference: reference("weather-plugin-a")}}
	if Validate(value) == nil {
		t.Fatal("unsorted plugins accepted")
	}
}
func TestReleaseDocumentURLIsDerivedFromIdentity(t *testing.T) {
	got := ReleaseDocumentURL(reference("weather-plugin"))
	want := "https://github.com/soksak-ai/weather-plugin/releases/download/v0.0.1/release.json"
	if got != want {
		t.Fatalf("derived %q, want %q", got, want)
	}
}
