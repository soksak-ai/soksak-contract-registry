package registry

import (
	"testing"

	composition "github.com/soksak-ai/soksak-contract-composition"
)

const commit = "0123456789abcdef0123456789abcdef01234567"
const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func pluginRelease(id string) PluginRelease {
	return PluginRelease{
		Spec:         ReleaseSpec,
		Plugin:       composition.PluginRef{ID: id, Version: "0.0.1"},
		Source:       Source{Repository: "https://github.com/example/" + id, Commit: commit},
		Dependencies: []Dependency{},
		Artifacts:    []Artifact{{Target: "any", URL: "https://example.invalid/p.tgz", SHA256: digest, Format: "tgz", Manifest: "plugin.json"}},
		Reports:      []Integrity{{URL: "https://example.invalid/report", SHA256: digest}},
	}
}

func sidecarRelease(id string) SidecarRelease {
	return SidecarRelease{
		Spec:         ReleaseSpec,
		Sidecar:      composition.SidecarRef{ID: id, Version: "0.0.1"},
		Source:       Source{Repository: "https://github.com/example/" + id, Commit: commit},
		Dependencies: []Dependency{},
		Artifacts:    []Artifact{{Target: "aarch64-apple-darwin", URL: "https://example.invalid/s.tgz", SHA256: digest, Format: "tgz", Manifest: "sidecar.json"}},
		Reports:      []Integrity{{URL: "https://example.invalid/report", SHA256: digest}},
	}
}

func TestRegistryKeepsPluginSidecarAndKitReleasesSeparate(t *testing.T) {
	plugin := pluginRelease("view")
	sidecar := sidecarRelease("state")
	plugin.Dependencies = []Dependency{{Sidecar: &sidecar.Sidecar, Scope: Runtime}}
	registry := Registry{
		Spec: RegistrySpec, ID: "official", Sequence: 1,
		Plugins: []PluginRelease{plugin}, Sidecars: []SidecarRelease{sidecar}, Kits: []KitRelease{},
		Profiles: []Profile{{
			ID: "view", Plugin: plugin.Plugin,
			Bindings: []composition.Binding{{
				Consumer: composition.Endpoint{Plugin: &plugin.Plugin}, Requirement: "state",
				Provider: composition.Endpoint{Sidecar: &sidecar.Sidecar},
			}},
		}},
	}
	if err := Validate(registry); err != nil {
		t.Fatal(err)
	}
}

func TestBuildDependencyDoesNotEnterProfileRuntimeClosure(t *testing.T) {
	plugin := pluginRelease("view")
	kit := KitRelease{
		Spec: ReleaseSpec, Kit: composition.KitRef{ID: "build", Version: "0.0.1"},
		Source:       Source{Repository: "https://github.com/example/build", Commit: commit},
		Dependencies: []Dependency{},
		Artifacts:    []Artifact{{Target: "any", URL: "https://example.invalid/k.tgz", SHA256: digest, Format: "tgz", Manifest: "package.json"}},
		Reports:      []Integrity{{URL: "https://example.invalid/report", SHA256: digest}},
	}
	plugin.Dependencies = []Dependency{{Kit: &kit.Kit, Scope: Build}}
	registry := Registry{
		Spec: RegistrySpec, ID: "official", Sequence: 1,
		Plugins: []PluginRelease{plugin}, Sidecars: []SidecarRelease{}, Kits: []KitRelease{kit},
		Profiles: []Profile{{ID: "view", Plugin: plugin.Plugin, Bindings: []composition.Binding{}}},
	}
	if err := Validate(registry); err != nil {
		t.Fatal(err)
	}
}
