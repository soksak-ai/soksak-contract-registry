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

func Sign(document *Registry, private ed25519.PrivateKey) error {
	if document.Signature.Algorithm != "ed25519" || document.Signature.KeyID == "" {
		return fmt.Errorf("invalid signature identity")
	}
	placeholder := document.Signature.Value
	document.Signature.Value = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if err := Validate(*document); err != nil {
		document.Signature.Value = placeholder
		return err
	}
	payload, err := signedPayload(*document)
	if err != nil {
		return err
	}
	document.Signature.Value = base64.StdEncoding.EncodeToString(ed25519.Sign(private, payload))
	return nil
}

func Verify(document Registry, trust Trust, now time.Time, highWater *HighWater) (Verification, error) {
	if err := Validate(document); err != nil {
		return Verification{}, err
	}
	if document.ID != trust.RegistryID || document.Signature.KeyID != trust.KeyID || document.Signature.Algorithm != "ed25519" || len(trust.PublicKey) != ed25519.PublicKeySize {
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
	signature, err := base64.StdEncoding.DecodeString(document.Signature.Value)
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

func signedPayload(document Registry) ([]byte, error) {
	value := struct {
		ID        string   `json:"id"`
		Sequence  uint64   `json:"sequence"`
		IssuedAt  string   `json:"issuedAt"`
		ExpiresAt string   `json:"expiresAt"`
		Plugins   []Plugin `json:"plugins"`
	}{document.ID, document.Sequence, document.IssuedAt, document.ExpiresAt, document.Plugins}
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
