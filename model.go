package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
)

type PluginReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type SidecarReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type KitReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type ContractReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
type SpecReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}
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
	Size     uint64 `json:"size"`
	SHA256   string `json:"sha256"`
	Format   string `json:"format"`
	Manifest string `json:"manifest"`
}
type PluginRelease struct {
	Plugin    PluginReference `json:"plugin"`
	Source    Source          `json:"source"`
	Artifacts []Artifact      `json:"artifacts"`
	Reports   []Integrity     `json:"reports"`
}
type SidecarRelease struct {
	Sidecar   SidecarReference `json:"sidecar"`
	Source    Source           `json:"source"`
	Artifacts []Artifact       `json:"artifacts"`
	Reports   []Integrity      `json:"reports"`
}
type KitRelease struct {
	Kit       KitReference `json:"kit"`
	Source    Source       `json:"source"`
	Artifacts []Artifact   `json:"artifacts"`
	Reports   []Integrity  `json:"reports"`
}
type ContractRelease struct {
	Contract  ContractReference `json:"contract"`
	Source    Source            `json:"source"`
	Artifacts []Artifact        `json:"artifacts"`
	Reports   []Integrity       `json:"reports"`
}
type SpecRelease struct {
	Spec      SpecReference `json:"spec"`
	Source    Source        `json:"source"`
	Artifacts []Artifact    `json:"artifacts"`
	Reports   []Integrity   `json:"reports"`
}
type Registry struct {
	ID        string            `json:"id"`
	Sequence  uint64            `json:"sequence"`
	Plugins   []PluginRelease   `json:"plugins"`
	Sidecars  []SidecarRelease  `json:"sidecars"`
	Kits      []KitRelease      `json:"kits"`
	Contracts []ContractRelease `json:"contracts"`
	Specs     []SpecRelease     `json:"specs"`
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
var registryPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func Parse(body []byte) (Registry, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var value Registry
	if err := decoder.Decode(&value); err != nil {
		return Registry{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Registry{}, fmt.Errorf("registry has trailing data")
	}
	if err := Validate(value); err != nil {
		return Registry{}, err
	}
	return value, nil
}

func Validate(value Registry) error {
	if !registryPattern.MatchString(value.ID) || value.Sequence < 1 {
		return fmt.Errorf("invalid registry identity")
	}
	if value.Plugins == nil || value.Sidecars == nil || value.Kits == nil || value.Contracts == nil || value.Specs == nil {
		return fmt.Errorf("all direct release arrays are required")
	}
	keys := []string{}
	for _, release := range value.Plugins {
		if err := validateRelease("plugin", release.Plugin.ID, release.Plugin.Version, release.Source, release.Artifacts, release.Reports, "plugin.json", false); err != nil {
			return err
		}
		keys = append(keys, "plugin:"+release.Plugin.ID+"@"+release.Plugin.Version)
	}
	for _, release := range value.Sidecars {
		if err := validateRelease("sidecar", release.Sidecar.ID, release.Sidecar.Version, release.Source, release.Artifacts, release.Reports, "sidecar.json", true); err != nil {
			return err
		}
		keys = append(keys, "sidecar:"+release.Sidecar.ID+"@"+release.Sidecar.Version)
	}
	for _, release := range value.Kits {
		if err := validateRelease("kit", release.Kit.ID, release.Kit.Version, release.Source, release.Artifacts, release.Reports, "kit.json", false); err != nil {
			return err
		}
		keys = append(keys, "kit:"+release.Kit.ID+"@"+release.Kit.Version)
	}
	for _, release := range value.Contracts {
		if err := validateRelease("contract", release.Contract.ID, release.Contract.Version, release.Source, release.Artifacts, release.Reports, "contract.json", false); err != nil {
			return err
		}
		keys = append(keys, "contract:"+release.Contract.ID+"@"+release.Contract.Version)
	}
	for _, release := range value.Specs {
		if err := validateRelease("spec", release.Spec.ID, release.Spec.Version, release.Source, release.Artifacts, release.Reports, "spec.json", false); err != nil {
			return err
		}
		keys = append(keys, "spec:"+release.Spec.ID+"@"+release.Spec.Version)
	}
	if !sortedUnique(keys) {
		return fmt.Errorf("releases must be globally sorted and unique")
	}
	return nil
}

func validateRelease(kind, id, version string, source Source, artifacts []Artifact, reports []Integrity, manifest string, native bool) error {
	key := kind + ":" + id + "@" + version
	if !idPattern.MatchString(id) || version != "0.0.1" {
		return fmt.Errorf("invalid release %s", key)
	}
	if source.Repository == "" || !commitPattern.MatchString(source.Commit) {
		return fmt.Errorf("invalid source %s", key)
	}
	if len(artifacts) == 0 || len(reports) == 0 {
		return fmt.Errorf("incomplete release %s", key)
	}
	targets := []string{}
	for _, artifact := range artifacts {
		if artifact.Target == "" || artifact.URL == "" || artifact.Size == 0 || !digestPattern.MatchString(artifact.SHA256) || (artifact.Format != "tgz" && artifact.Format != "tar.gz") || artifact.Manifest != manifest {
			return fmt.Errorf("invalid artifact %s", key)
		}
		if !native && artifact.Target != "any" {
			return fmt.Errorf("portable release has native target %s", key)
		}
		targets = append(targets, artifact.Target)
	}
	if !sortedUnique(targets) {
		return fmt.Errorf("artifacts must be sorted and unique")
	}
	urls := []string{}
	for _, report := range reports {
		if report.URL == "" || !digestPattern.MatchString(report.SHA256) {
			return fmt.Errorf("invalid report %s", key)
		}
		urls = append(urls, report.URL)
	}
	if !sortedUnique(urls) {
		return fmt.Errorf("reports must be sorted and unique")
	}
	return nil
}

func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return false
		}
	}
	return true
}
