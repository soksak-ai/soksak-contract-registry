package registry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"
)

type ReleaseReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	URL     string `json:"url"`
	Size    uint64 `json:"size"`
	SHA256  string `json:"sha256"`
}
type RuntimeDependencies struct {
	Plugins  []ReleaseReference `json:"plugins,omitempty"`
	Sidecars []ReleaseReference `json:"sidecars,omitempty"`
}
type Plugin struct {
	ReleaseReference
	RuntimeDependencies *RuntimeDependencies `json:"runtimeDependencies,omitempty"`
}
type Signature struct {
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Value     string `json:"value"`
}
type Registry struct {
	ID        string    `json:"id"`
	Sequence  uint64    `json:"sequence"`
	IssuedAt  string    `json:"issuedAt"`
	ExpiresAt string    `json:"expiresAt"`
	Plugins   []Plugin  `json:"plugins"`
	Signature Signature `json:"signature"`
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
var registryPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)
var releaseURLPattern = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9-]+/([A-Za-z0-9._-]+)/releases/download/v([^/]+)/release\.json$`)

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
	issued, err := time.Parse(time.RFC3339, value.IssuedAt)
	if err != nil {
		return fmt.Errorf("invalid issuedAt")
	}
	expires, err := time.Parse(time.RFC3339, value.ExpiresAt)
	if err != nil || !expires.After(issued) {
		return fmt.Errorf("invalid expiresAt")
	}
	if value.Plugins == nil {
		return fmt.Errorf("plugins are required")
	}
	ids := make([]string, 0, len(value.Plugins))
	for _, plugin := range value.Plugins {
		if err := validateReference(plugin.ReleaseReference); err != nil {
			return err
		}
		ids = append(ids, plugin.ID)
		if plugin.RuntimeDependencies != nil {
			if err := validateReferences(plugin.RuntimeDependencies.Plugins, "plugin"); err != nil {
				return err
			}
			if err := validateReferences(plugin.RuntimeDependencies.Sidecars, "sidecar"); err != nil {
				return err
			}
			if len(plugin.RuntimeDependencies.Plugins) == 0 && len(plugin.RuntimeDependencies.Sidecars) == 0 {
				return fmt.Errorf("empty runtimeDependencies")
			}
		}
	}
	if !sortedUnique(ids) {
		return fmt.Errorf("plugins must be sorted and unique by id")
	}
	decoded, err := base64.StdEncoding.DecodeString(value.Signature.Value)
	if value.Signature.Algorithm != "ed25519" || value.Signature.KeyID == "" || err != nil || len(decoded) != 64 {
		return fmt.Errorf("invalid registry signature shape")
	}
	return nil
}
func validateReferences(values []ReleaseReference, kind string) error {
	keys := make([]string, 0, len(values))
	for _, value := range values {
		if err := validateReference(value); err != nil {
			return err
		}
		keys = append(keys, value.ID+"@"+value.Version)
	}
	if len(values) > 0 && !sortedUnique(keys) {
		return fmt.Errorf("%s dependencies must be sorted and unique", kind)
	}
	return nil
}
func validateReference(value ReleaseReference) error {
	match := releaseURLPattern.FindStringSubmatch(value.URL)
	if !idPattern.MatchString(value.ID) || !semverPattern.MatchString(value.Version) || value.Size == 0 || !digestPattern.MatchString(value.SHA256) || len(match) != 3 || match[1] != value.ID || match[2] != value.Version {
		return fmt.Errorf("invalid release reference %s@%s", value.ID, value.Version)
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
