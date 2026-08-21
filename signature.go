package registry

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type SignedRegistry struct {
	Registry
	IssuedAt  string `json:"issuedAt"`
	ExpiresAt string `json:"expiresAt"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"keyId"`
	Signature string `json:"signature"`
}

type Trust struct {
	RegistryID string
	KeyID      string
	PublicKey  ed25519.PublicKey
}
type HighWater struct {
	Sequence uint64 `json:"sequence"`
	Digest   string `json:"digest"`
}
type Verification struct {
	Sequence   uint64 `json:"sequence"`
	Digest     string `json:"digest"`
	Continuity string `json:"continuity"`
}

func Sign(document *SignedRegistry, private ed25519.PrivateKey) error {
	if document.Algorithm != "ed25519" || document.KeyID == "" {
		return fmt.Errorf("invalid signature identity")
	}
	if err := Validate(document.Registry); err != nil {
		return err
	}
	payload, err := signedPayload(*document)
	if err != nil {
		return err
	}
	document.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(private, payload))
	return nil
}

func Verify(document SignedRegistry, trust Trust, now time.Time, highWater *HighWater) (Verification, error) {
	if err := Validate(document.Registry); err != nil {
		return Verification{}, err
	}
	if document.ID != trust.RegistryID || document.KeyID != trust.KeyID || document.Algorithm != "ed25519" || len(trust.PublicKey) != ed25519.PublicKeySize {
		return Verification{}, fmt.Errorf("registry trust identity mismatch")
	}
	issued, err := time.Parse(time.RFC3339, document.IssuedAt)
	if err != nil {
		return Verification{}, fmt.Errorf("invalid issuedAt")
	}
	expires, err := time.Parse(time.RFC3339, document.ExpiresAt)
	if err != nil || !expires.After(issued) {
		return Verification{}, fmt.Errorf("invalid expiresAt")
	}
	if now.Before(issued) || !now.Before(expires) {
		return Verification{}, fmt.Errorf("registry is not current")
	}
	payload, err := signedPayload(document)
	if err != nil {
		return Verification{}, err
	}
	signature, err := base64.StdEncoding.DecodeString(document.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(trust.PublicKey, payload, signature) {
		return Verification{}, fmt.Errorf("registry signature is invalid")
	}
	digestRaw := sha256.Sum256(payload)
	digest := hex.EncodeToString(digestRaw[:])
	continuity := "initial"
	if highWater != nil {
		if document.Sequence < highWater.Sequence {
			return Verification{}, fmt.Errorf("registry sequence rollback")
		}
		if document.Sequence == highWater.Sequence {
			if digest != highWater.Digest {
				return Verification{}, fmt.Errorf("registry sequence equivocation")
			}
			continuity = "unchanged"
		} else {
			continuity = "advance"
		}
	}
	return Verification{Sequence: document.Sequence, Digest: digest, Continuity: continuity}, nil
}

func signedPayload(document SignedRegistry) ([]byte, error) {
	value := struct {
		Registry  Registry `json:"registry"`
		IssuedAt  string   `json:"issuedAt"`
		ExpiresAt string   `json:"expiresAt"`
		Algorithm string   `json:"algorithm"`
		KeyID     string   `json:"keyId"`
	}{document.Registry, document.IssuedAt, document.ExpiresAt, document.Algorithm, document.KeyID}
	return canonicalJSON(value)
}
func canonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := writeCanonical(&output, decoded); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
func writeCanonical(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool, string, json.Number:
		raw, _ := json.Marshal(typed)
		output.Write(raw)
	case []any:
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeCanonical(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			raw, _ := json.Marshal(key)
			output.Write(raw)
			output.WriteByte(':')
			if err := writeCanonical(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value")
	}
	return nil
}
