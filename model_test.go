package registry

import "testing"

const commit = "0123456789abcdef0123456789abcdef01234567"
const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func pluginRelease(id string, size uint64) PluginRelease {
	repository := "https://github.com/example/" + id
	return PluginRelease{
		Plugin:    PluginReference{ID: id, Version: "0.0.1"},
		Source:    Source{Repository: repository, Commit: commit},
		Artifacts: []Artifact{{Target: "any", URL: repository + "/releases/download/v0.0.1/view.tgz", Size: size, SHA256: digest, Format: "tgz", Manifest: "plugin.json"}},
		Reports:   []Integrity{{URL: repository + "/releases/download/v0.0.1/report.json", SHA256: digest}},
	}
}

func TestRegistryAcceptsAPatchSpecReleaseAndBindsItsTag(t *testing.T) {
	repository := "https://github.com/soksak-ai/soksak-spec"
	release := SpecRelease{
		Spec:   SpecReference{ID: "soksak-spec", Version: "0.0.2"},
		Source: Source{Repository: repository, Commit: commit},
		Artifacts: []Artifact{{
			Target: "any", URL: repository + "/releases/download/v0.0.2/soksak-ai-plugin-spec-0.0.2.tgz",
			Size: 10, SHA256: digest, Format: "tgz", Manifest: "spec.json",
		}},
		Reports: []Integrity{{URL: repository + "/releases/download/v0.0.2/conformance-release.json", SHA256: digest}},
	}
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{release}}
	if err := Validate(value); err != nil {
		t.Fatal(err)
	}
	value.Specs[0].Artifacts[0].URL = repository + "/releases/download/v0.0.1/soksak-ai-plugin-spec-0.0.2.tgz"
	if err := Validate(value); err == nil {
		t.Fatal("release asset tag does not match the release version")
	}
}

func TestRegistryRejectsVersionLocators(t *testing.T) {
	for _, version := range []string{"^0.0.1", "latest", "0.0"} {
		release := pluginRelease("view", 10)
		release.Plugin.Version = version
		value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{release}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
		if err := Validate(value); err == nil {
			t.Errorf("accepted version %q", version)
		}
	}
}

func TestRegistryAcceptsFiveDirectReleaseArrays(t *testing.T) {
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{pluginRelease("view", 10)}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryAcceptsMultipleDirectReleaseKinds(t *testing.T) {
	plugin := pluginRelease("view", 10)
	repository := "https://github.com/example/state"
	sidecar := SidecarRelease{
		Sidecar:   SidecarReference{ID: "state", Version: "0.0.1"},
		Source:    Source{Repository: repository, Commit: commit},
		Artifacts: []Artifact{{Target: "x86_64-pc-windows-msvc", URL: repository + "/releases/download/v0.0.1/state.tar.gz", Size: 10, SHA256: digest, Format: "tar.gz", Manifest: "sidecar.json"}},
		Reports:   []Integrity{{URL: repository + "/releases/download/v0.0.1/report.json", SHA256: digest}},
	}
	kitRepository := "https://github.com/example/kit"
	kit := KitRelease{
		Kit:       KitReference{ID: "kit", Version: "0.0.1"},
		Source:    Source{Repository: kitRepository, Commit: commit},
		Artifacts: []Artifact{{Target: "any", URL: kitRepository + "/releases/download/v0.0.1/kit.tgz", Size: 10, SHA256: digest, Format: "tgz", Manifest: "kit.json"}},
		Reports:   []Integrity{{URL: kitRepository + "/releases/download/v0.0.1/report.json", SHA256: digest}},
	}
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{plugin}, Sidecars: []SidecarRelease{sidecar}, Kits: []KitRelease{kit}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err != nil {
		t.Fatal(err)
	}
}

func TestRegistryRejectsUnsortedReleasesWithinTheirKind(t *testing.T) {
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{pluginRelease("z-view", 10), pluginRelease("a-view", 10)}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err == nil {
		t.Fatal("unsorted plugin releases were accepted")
	}
}

func TestRegistryRejectsReleaseHistoryForOneComponent(t *testing.T) {
	older := pluginRelease("view", 10)
	newer := pluginRelease("view", 10)
	newer.Plugin.Version = "0.0.2"
	newer.Artifacts[0].URL = "https://github.com/example/view/releases/download/v0.0.2/view.tgz"
	newer.Reports[0].URL = "https://github.com/example/view/releases/download/v0.0.2/report.json"
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{older, newer}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err == nil {
		t.Fatal("registry accepted two current releases for one component id")
	}
}

func TestRegistryRejectsArtifactWithoutSize(t *testing.T) {
	value := Registry{ID: "official", Sequence: 1, Plugins: []PluginRelease{pluginRelease("view", 0)}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{}, Contracts: []ContractRelease{}, Specs: []SpecRelease{}}
	if err := Validate(value); err == nil {
		t.Fatal("artifact without size was accepted")
	}
}
