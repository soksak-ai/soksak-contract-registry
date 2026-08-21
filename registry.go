package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"

	composition "github.com/soksak-ai/soksak-contract-composition"
)

const (
	RegistrySpec = "soksak-spec-registry@0.0.1"
	ReleaseSpec  = "soksak-spec-release@0.0.1"
)

type DependencyScope string

const (
	Runtime DependencyScope = "runtime"
	Build   DependencyScope = "build"
)

type Source struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
}
type Integrity struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}
type Artifact struct {
	Target   string `json:"target"`
	URL      string `json:"url"`
	SHA256   string `json:"sha256"`
	Format   string `json:"format"`
	Manifest string `json:"manifest"`
}
type Dependency struct {
	Plugin  *composition.PluginRef  `json:"plugin,omitempty"`
	Sidecar *composition.SidecarRef `json:"sidecar,omitempty"`
	Kit     *composition.KitRef     `json:"kit,omitempty"`
	Scope   DependencyScope         `json:"scope"`
}
type releaseFields struct {
	Spec         string
	Source       Source
	Dependencies []Dependency
	Artifacts    []Artifact
	Reports      []Integrity
}
type PluginRelease struct {
	Spec         string                `json:"spec"`
	Plugin       composition.PluginRef `json:"plugin"`
	Source       Source                `json:"source"`
	Dependencies []Dependency          `json:"dependencies"`
	Artifacts    []Artifact            `json:"artifacts"`
	Reports      []Integrity           `json:"reports"`
}
type SidecarRelease struct {
	Spec         string                 `json:"spec"`
	Sidecar      composition.SidecarRef `json:"sidecar"`
	Source       Source                 `json:"source"`
	Dependencies []Dependency           `json:"dependencies"`
	Artifacts    []Artifact             `json:"artifacts"`
	Reports      []Integrity            `json:"reports"`
}
type KitRelease struct {
	Spec         string             `json:"spec"`
	Kit          composition.KitRef `json:"kit"`
	Source       Source             `json:"source"`
	Dependencies []Dependency       `json:"dependencies"`
	Artifacts    []Artifact         `json:"artifacts"`
	Reports      []Integrity        `json:"reports"`
}
type Profile struct {
	ID       string                `json:"id"`
	Plugin   composition.PluginRef `json:"plugin"`
	Bindings []composition.Binding `json:"bindings"`
}
type Registry struct {
	Spec     string           `json:"spec"`
	ID       string           `json:"id"`
	Sequence uint64           `json:"sequence"`
	Plugins  []PluginRelease  `json:"plugins"`
	Sidecars []SidecarRelease `json:"sidecars"`
	Kits     []KitRelease     `json:"kits"`
	Profiles []Profile        `json:"profiles"`
}

var idPattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]*$")
var versionPattern = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$")
var commitPattern = regexp.MustCompile("^[0-9a-f]{40}$")
var digestPattern = regexp.MustCompile("^[0-9a-f]{64}$")

func Parse(body []byte) (Registry, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var registry Registry
	if err := decoder.Decode(&registry); err != nil {
		return Registry{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Registry{}, fmt.Errorf("registry has trailing data")
	}
	normalize(&registry)
	if err := Validate(registry); err != nil {
		return Registry{}, err
	}
	return registry, nil
}
func Validate(registry Registry) error {
	if registry.Spec != RegistrySpec || !idPattern.MatchString(registry.ID) || registry.Sequence < 1 {
		return fmt.Errorf("invalid registry identity")
	}
	if registry.Plugins == nil || registry.Sidecars == nil || registry.Kits == nil || registry.Profiles == nil {
		return fmt.Errorf("plugins, sidecars, kits and profiles arrays are required")
	}
	index := map[string][]Dependency{}
	pluginKeys := []string{}
	sidecarKeys := []string{}
	kitKeys := []string{}
	for _, release := range registry.Plugins {
		key := "plugin:" + release.Plugin.ID + "@" + release.Plugin.Version
		if err := validateRelease(key, release.Spec, release.Plugin.ID, release.Plugin.Version, release.Source, release.Dependencies, release.Artifacts, release.Reports, "plugin.json"); err != nil {
			return err
		}
		index[key] = release.Dependencies
		pluginKeys = append(pluginKeys, key)
	}
	for _, release := range registry.Sidecars {
		key := "sidecar:" + release.Sidecar.ID + "@" + release.Sidecar.Version
		if err := validateRelease(key, release.Spec, release.Sidecar.ID, release.Sidecar.Version, release.Source, release.Dependencies, release.Artifacts, release.Reports, "sidecar.json"); err != nil {
			return err
		}
		index[key] = release.Dependencies
		sidecarKeys = append(sidecarKeys, key)
	}
	for _, release := range registry.Kits {
		key := "kit:" + release.Kit.ID + "@" + release.Kit.Version
		if err := validateRelease(key, release.Spec, release.Kit.ID, release.Kit.Version, release.Source, release.Dependencies, release.Artifacts, release.Reports, "package.json"); err != nil {
			return err
		}
		index[key] = release.Dependencies
		kitKeys = append(kitKeys, key)
	}
	if !sortedUnique(pluginKeys) || !sortedUnique(sidecarKeys) || !sortedUnique(kitKeys) {
		return fmt.Errorf("releases must be sorted and unique")
	}
	profileIDs := []string{}
	for _, profile := range registry.Profiles {
		root := "plugin:" + profile.Plugin.ID + "@" + profile.Plugin.Version
		if _, exists := index[root]; !exists {
			return fmt.Errorf("profile plugin is absent")
		}
		profileClosure, err := closure(root, index)
		if err != nil {
			return err
		}
		for _, binding := range profile.Bindings {
			consumer, err := endpointKey(binding.Consumer)
			if err != nil {
				return err
			}
			if !profileClosure[consumer] {
				return fmt.Errorf("profile binding consumer leaves plugin closure")
			}
			provider, err := endpointKey(binding.Provider)
			if err != nil {
				return err
			}
			providerClosure, err := closure(provider, index)
			if err != nil {
				return err
			}
			for key := range providerClosure {
				profileClosure[key] = true
			}
		}
		profileIDs = append(profileIDs, profile.ID)
	}
	if !sortedUnique(profileIDs) {
		return fmt.Errorf("profiles must be sorted and unique")
	}
	return nil
}

func validateRelease(key, spec, id, version string, source Source, dependencies []Dependency, artifacts []Artifact, reports []Integrity, manifest string) error {
	if spec != ReleaseSpec || !idPattern.MatchString(id) || !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid release %s", key)
	}
	if source.Repository == "" || !commitPattern.MatchString(source.Commit) {
		return fmt.Errorf("invalid source %s", key)
	}
	if dependencies == nil || len(artifacts) == 0 || len(reports) == 0 {
		return fmt.Errorf("incomplete release %s", key)
	}
	dependencyKeys := []string{}
	for _, dependency := range dependencies {
		dependencyKey, err := dependencyKey(dependency)
		if err != nil {
			return err
		}
		dependencyKeys = append(dependencyKeys, dependencyKey)
	}
	if !sortedUnique(dependencyKeys) {
		return fmt.Errorf("dependencies must be sorted and unique")
	}
	targets := []string{}
	for _, artifact := range artifacts {
		if artifact.Target == "" || artifact.URL == "" || !digestPattern.MatchString(artifact.SHA256) || (artifact.Format != "tgz" && artifact.Format != "tar.gz") || artifact.Manifest != manifest {
			return fmt.Errorf("invalid artifact %s", key)
		}
		targets = append(targets, artifact.Target)
	}
	if !sortedUnique(targets) {
		return fmt.Errorf("artifacts must be sorted and unique")
	}
	reportURLs := []string{}
	for _, report := range reports {
		if report.URL == "" || !digestPattern.MatchString(report.SHA256) {
			return fmt.Errorf("invalid report %s", key)
		}
		reportURLs = append(reportURLs, report.URL)
	}
	if !sortedUnique(reportURLs) {
		return fmt.Errorf("reports must be sorted and unique")
	}
	return nil
}
func dependencyKey(value Dependency) (string, error) {
	if value.Scope != Runtime && value.Scope != Build {
		return "", fmt.Errorf("dependency scope required")
	}
	endpoint := composition.Endpoint{Plugin: value.Plugin, Sidecar: value.Sidecar, Kit: value.Kit}
	key, err := endpointKey(endpoint)
	if err != nil {
		return "", err
	}
	return key, nil
}
func endpointKey(value composition.Endpoint) (string, error) {
	count := 0
	key := ""
	if value.Plugin != nil {
		count++
		key = "plugin:" + value.Plugin.ID + "@" + value.Plugin.Version
	}
	if value.Sidecar != nil {
		count++
		key = "sidecar:" + value.Sidecar.ID + "@" + value.Sidecar.Version
	}
	if value.Kit != nil {
		count++
		key = "kit:" + value.Kit.ID + "@" + value.Kit.Version
	}
	if count != 1 {
		return "", fmt.Errorf("exactly one plugin, sidecar or kit reference required")
	}
	return key, nil
}
func closure(root string, index map[string][]Dependency) (map[string]bool, error) {
	result := map[string]bool{}
	visiting := map[string]bool{}
	var visit func(string) error
	visit = func(key string) error {
		if result[key] {
			return nil
		}
		if visiting[key] {
			return fmt.Errorf("dependency cycle")
		}
		dependencies, exists := index[key]
		if !exists {
			return fmt.Errorf("dependency absent: %s", key)
		}
		visiting[key] = true
		for _, dependency := range dependencies {
			if dependency.Scope == Build {
				continue
			}
			next, err := dependencyKey(dependency)
			if err != nil {
				return err
			}
			if err := visit(next); err != nil {
				return err
			}
		}
		visiting[key] = false
		result[key] = true
		return nil
	}
	return result, visit(root)
}
func normalize(registry *Registry) {
	if registry.Plugins == nil {
		registry.Plugins = []PluginRelease{}
	}
	if registry.Sidecars == nil {
		registry.Sidecars = []SidecarRelease{}
	}
	if registry.Kits == nil {
		registry.Kits = []KitRelease{}
	}
	if registry.Profiles == nil {
		registry.Profiles = []Profile{}
	}
}
func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for i := 1; i < len(values); i++ {
		if values[i-1] == values[i] {
			return false
		}
	}
	return true
}
