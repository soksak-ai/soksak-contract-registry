package registry

import "testing"

const commit = "0123456789abcdef0123456789abcdef01234567"
const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func pluginRelease(id string, size uint64) PluginRelease {
	return PluginRelease{
		Plugin:    PluginReference{ID: id, Version: "0.0.1"},
		Source:    Source{Repository: "https://github.com/example/" + id, Commit: commit},
		Artifacts: []Artifact{{Target: "any", URL: "https://example.invalid/view.tgz", Size: size, SHA256: digest, Format: "tgz", Manifest: "plugin.json"}},
		Reports:   []Integrity{{URL: "https://example.invalid/report", SHA256: digest}},
	}
}

func TestRegistryAcceptsFiveDirectReleaseArrays(t *testing.T) {
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{pluginRelease("view", 10)}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryRejectsArtifactWithoutSize(t *testing.T) {
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{pluginRelease("view", 0)}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err == nil {
		t.Fatal("artifact without size was accepted")
	}
}
