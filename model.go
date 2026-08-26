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

// GitHubOrg is the organization every release is published under.
const GitHubOrg = "soksak-ai"

// ReleaseReference pins one release; Size and SHA256 are of that release's
// release.json. No reference carries a location: the reader derives it with
// ReleaseDocumentURL. This module does not depend on platformspec, so the
// reference shape and the id, version and digest grammars below restate
// soksak-spec/go/platformspec/release.go once.
type ReleaseReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

// Plugin is one current release reference. The index copies no runtimeDependencies: a reader walks
// the closure from the release document the reference names.
type Plugin struct {
	ReleaseReference
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
var strictSemverPattern = regexp.MustCompile(`^(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

const maxSemverLength = 256

// ReleaseDocumentURL derives the published release.json location from the
// identity: https://github.com/soksak-ai/<id>/releases/download/v<version>/release.json.
func ReleaseDocumentURL(value ReleaseReference) string {
	return "https://github.com/" + GitHubOrg + "/" + value.ID + "/releases/download/v" + value.Version + "/release.json"
}

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
	if !idPattern.MatchString(value.ID) || !strictSemver(value.Version) || value.Size <= 0 || !digestPattern.MatchString(value.SHA256) {
		return fmt.Errorf("invalid release reference %s@%s", value.ID, value.Version)
	}
	return nil
}
func strictSemver(value string) bool {
	return len(value) > 0 && len(value) <= maxSemverLength && strictSemverPattern.MatchString(value)
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
