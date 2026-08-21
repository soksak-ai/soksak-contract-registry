package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	composition "github.com/soksak/soksak-contract-composition"
)

const (
	RegistrySpec = "soksak-spec-registry@0.0.1"
	ReleaseSpec  = "soksak-spec-release@0.0.1"
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
	Target       string `json:"target"`
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	Format       string `json:"format"`
	UnitManifest string `json:"unitManifest"`
}
type Release struct {
	Spec         string                `json:"spec"`
	Unit         composition.UnitRef   `json:"unit"`
	Source       Source                `json:"source"`
	Dependencies []composition.UnitRef `json:"dependencies"`
	Artifacts    []Artifact            `json:"artifacts"`
	Reports      []Integrity           `json:"reports"`
}
type Profile struct {
	ID       string                `json:"id"`
	Root     composition.UnitRef   `json:"root"`
	Bindings []composition.Binding `json:"bindings"`
}
type Registry struct {
	Spec     string    `json:"spec"`
	ID       string    `json:"id"`
	Sequence uint64    `json:"sequence"`
	Releases []Release `json:"releases"`
	Profiles []Profile `json:"profiles"`
}

var exactVersion = regexp.MustCompile("^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$")
var commitPattern = regexp.MustCompile("^[0-9a-f]{40}$")
var digestPattern = regexp.MustCompile("^[0-9a-f]{64}$")
var idPattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]*$")

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
	if err := Validate(registry); err != nil {
		return Registry{}, err
	}
	return registry, nil
}

func Validate(registry Registry) error {
	if registry.Spec != RegistrySpec || !idPattern.MatchString(registry.ID) || registry.Sequence < 1 {
		return fmt.Errorf("invalid registry identity")
	}
	if registry.Releases == nil || registry.Profiles == nil {
		return fmt.Errorf("releases and profiles arrays are required")
	}
	releases := make(map[string]Release, len(registry.Releases))
	keys := make([]string, 0, len(registry.Releases))
	for _, release := range registry.Releases {
		if err := ValidateRelease(release); err != nil {
			return err
		}
		key := release.Unit.Key()
		if _, exists := releases[key]; exists {
			return fmt.Errorf("duplicate release %s", key)
		}
		releases[key] = release
		keys = append(keys, key)
	}
	if !sort.StringsAreSorted(keys) {
		return fmt.Errorf("releases must be sorted")
	}
	profileIDs := []string{}
	for _, profile := range registry.Profiles {
		if !idPattern.MatchString(profile.ID) || profile.Root.Kind != composition.Plugin {
			return fmt.Errorf("profile root must be a plugin")
		}
		if _, exists := releases[profile.Root.Key()]; !exists {
			return fmt.Errorf("profile root is absent")
		}
		closure, err := resolveClosure(profile.Root, releases)
		if err != nil {
			return err
		}
		for _, binding := range profile.Bindings {
			if !closure[binding.Consumer.Key()] || !closure[binding.Provider.Key()] {
				return fmt.Errorf("profile binding leaves closure")
			}
		}
		profileIDs = append(profileIDs, profile.ID)
	}
	if !sortedUnique(profileIDs) {
		return fmt.Errorf("profiles must be sorted and unique")
	}
	return nil
}

func ValidateRelease(release Release) error {
	if release.Spec != ReleaseSpec || !validUnit(release.Unit) {
		return fmt.Errorf("invalid release identity")
	}
	if release.Source.Repository == "" || !commitPattern.MatchString(release.Source.Commit) {
		return fmt.Errorf("release source requires repository and exact commit")
	}
	if release.Dependencies == nil || len(release.Artifacts) == 0 || len(release.Reports) == 0 {
		return fmt.Errorf("release dependencies, artifacts and reports are required")
	}
	dependencyKeys := []string{}
	for _, dependency := range release.Dependencies {
		if !validUnit(dependency) {
			return fmt.Errorf("invalid dependency")
		}
		dependencyKeys = append(dependencyKeys, dependency.Key())
	}
	if !sortedUnique(dependencyKeys) {
		return fmt.Errorf("dependencies must be sorted and unique")
	}
	targets := []string{}
	for _, artifact := range release.Artifacts {
		if artifact.Target == "" || artifact.URL == "" || !digestPattern.MatchString(artifact.SHA256) || (artifact.Format != "tgz" && artifact.Format != "tar.gz") || artifact.UnitManifest != composition.UnitManifestFile {
			return fmt.Errorf("invalid release artifact")
		}
		targets = append(targets, artifact.Target)
	}
	if !sortedUnique(targets) {
		return fmt.Errorf("artifact targets must be sorted and unique")
	}
	reports := []string{}
	for _, report := range release.Reports {
		if report.URL == "" || !digestPattern.MatchString(report.SHA256) {
			return fmt.Errorf("invalid conformance report")
		}
		reports = append(reports, report.URL)
	}
	if !sortedUnique(reports) {
		return fmt.Errorf("reports must be sorted and unique")
	}
	return nil
}

func resolveClosure(root composition.UnitRef, releases map[string]Release) (map[string]bool, error) {
	result := map[string]bool{}
	visiting := map[string]bool{}
	var visit func(composition.UnitRef) error
	visit = func(unit composition.UnitRef) error {
		key := unit.Key()
		if result[key] {
			return nil
		}
		if visiting[key] {
			return fmt.Errorf("dependency cycle")
		}
		release, exists := releases[key]
		if !exists {
			return fmt.Errorf("dependency absent from registry: %s", key)
		}
		visiting[key] = true
		for _, dependency := range release.Dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[key] = false
		result[key] = true
		return nil
	}
	return result, visit(root)
}
func validUnit(unit composition.UnitRef) bool {
	return (unit.Kind == composition.Plugin || unit.Kind == composition.Sidecar || unit.Kind == composition.Kit) && idPattern.MatchString(unit.ID) && exactVersion.MatchString(unit.Version)
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

func ProfileBindings(registry Registry, profileID string) ([]composition.Binding, error) {
	if err := Validate(registry); err != nil {
		return nil, err
	}
	for _, profile := range registry.Profiles {
		if profile.ID == profileID {
			return append([]composition.Binding(nil), profile.Bindings...), nil
		}
	}
	return nil, fmt.Errorf("profile not found: %s", strings.TrimSpace(profileID))
}
