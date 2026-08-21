package registry

import (
	"strings"
	"testing"

	composition "github.com/soksak/soksak-contract-composition"
)

const commit = "0123456789abcdef0123456789abcdef01234567"
const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func ref(kind composition.UnitKind, id string) composition.UnitRef {
	return composition.UnitRef{Kind: kind, ID: id, Version: "0.0.1"}
}

func release(unit composition.UnitRef) Release {
	return Release{
		Spec: ReleaseSpec, Unit: unit, Source: Source{Repository: "https://github.com/example/" + unit.ID, Commit: commit},
		Dependencies: []composition.UnitRef{},
		Artifacts:    []Artifact{{Target: "aarch64-apple-darwin", URL: "https://github.com/example/" + unit.ID + "/releases/download/v0.0.1/" + unit.ID + ".tgz", SHA256: digest, Format: "tgz", UnitManifest: composition.UnitManifestFile}},
		Reports:      []Integrity{{URL: "https://github.com/example/" + unit.ID + "/releases/download/v0.0.1/conformance.json", SHA256: digest}},
	}
}

func TestRegistryPublishesPluginsAndTheirExactClosure(t *testing.T) {
	plugin := ref(composition.Plugin, "terminal-view")
	provider := ref(composition.Sidecar, "terminal-state")
	kit := ref(composition.Kit, "terminal-runtime")
	pluginRelease := release(plugin)
	pluginRelease.Dependencies = []composition.UnitRef{kit, provider}
	registry := Registry{Spec: RegistrySpec, ID: "official", Sequence: 1, Releases: []Release{release(kit), pluginRelease, release(provider)}, Profiles: []Profile{{ID: "terminal-view", Root: plugin, Bindings: []composition.Binding{{Consumer: plugin, Requirement: "state", Provider: provider}}}}}
	if err := Validate(registry); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyPluginsCanBeProfileRoots(t *testing.T) {
	sidecar := ref(composition.Sidecar, "state")
	registry := Registry{Spec: RegistrySpec, ID: "official", Sequence: 1, Releases: []Release{release(sidecar)}, Profiles: []Profile{{ID: "bad", Root: sidecar, Bindings: []composition.Binding{}}}}
	if err := Validate(registry); err == nil {
		t.Fatal("sidecar profile root was accepted")
	}
}

func TestProfileBindingsStayInsideExactClosure(t *testing.T) {
	plugin := ref(composition.Plugin, "view")
	provider := ref(composition.Sidecar, "provider")
	other := ref(composition.Sidecar, "other")
	pluginRelease := release(plugin)
	pluginRelease.Dependencies = []composition.UnitRef{provider}
	registry := Registry{Spec: RegistrySpec, ID: "official", Sequence: 1, Releases: []Release{pluginRelease, release(other), release(provider)}, Profiles: []Profile{{ID: "view", Root: plugin, Bindings: []composition.Binding{{Consumer: plugin, Requirement: "state", Provider: other}}}}}
	if err := Validate(registry); err == nil || !strings.Contains(err.Error(), "closure") {
		t.Fatalf("error = %v", err)
	}
}

func TestReleaseRequiresCompositionManifestAndConformance(t *testing.T) {
	plugin := ref(composition.Plugin, "view")
	cases := []Release{release(plugin), release(plugin)}
	cases[0].Artifacts[0].UnitManifest = "plugin.json"
	cases[1].Reports = nil
	for _, candidate := range cases {
		if err := ValidateRelease(candidate); err == nil {
			t.Errorf("accepted invalid release: %+v", candidate)
		}
	}
}

func TestStrictParserRejectsFallbackAndUnknownFields(t *testing.T) {
	raw := "{\"spec\":\"soksak-spec-registry@0.0.1\",\"id\":\"official\",\"sequence\":1,\"releases\":[],\"profiles\":[],\"fallback\":true}"
	if _, err := Parse([]byte(raw)); err == nil {
		t.Fatal("registry accepted fallback")
	}
}
