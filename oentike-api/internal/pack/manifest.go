// Package pack signs and verifies immutable offline pack inventories.
// It does not activate packs or validate the contents of geometry/map formats.
package pack

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	Domain                 = "OENTIKE-PACK-MANIFEST-V1\x00"
	MaxManifestBytes       = 1 << 20
	MaxFiles               = 4096
	MaxFileBytes     int64 = 8 << 30
	MaxTotalBytes    int64 = 32 << 30
)

type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	AreaID        string `json:"area_id"`
	Release       uint64 `json:"release"`
	CreatedAt     string `json:"created_at"`
	Files         []File `json:"files"`
}

type File struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
	Format  string `json:"format"`
	Source  string `json:"source"`
	License string `json:"license"`
}

// Parse validates untrusted metadata, without establishing authenticity.
func Parse(data []byte) (Manifest, error) {
	var m Manifest
	if len(data) == 0 || len(data) > MaxManifestBytes {
		return m, fmt.Errorf("manifest size outside limit")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := checkJSON(d, 0); err != nil {
		return m, err
	}
	if _, err := d.Token(); err != io.EOF {
		return m, fmt.Errorf("trailing JSON")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return m, err
	}
	return m, m.Validate()
}

// checkJSON enforces exact field names, required fields, no duplicates/nulls,
// bounded nesting, and printable ASCII strings for cross-language agreement.
func checkJSON(d *json.Decoder, depth int) error {
	if depth > 3 {
		return fmt.Errorf("JSON nesting exceeds schema")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	switch v := t.(type) {
	case json.Delim:
		switch v {
		case '{':
			var fields string
			if depth == 0 {
				fields = "schema_version area_id release created_at files"
			} else if depth == 2 {
				fields = "path size sha256 format source license"
			} else {
				return fmt.Errorf("unexpected object")
			}
			allowed := strings.Fields(fields)
			seen := make(map[string]bool, len(allowed))
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("invalid or duplicate JSON key")
				}
				found := false
				for _, field := range allowed {
					if name == field {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("unknown JSON key %q", name)
				}
				seen[name] = true
				if err := checkJSON(d, depth+1); err != nil {
					return err
				}
			}
			if len(seen) != len(allowed) {
				return fmt.Errorf("missing JSON field")
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return fmt.Errorf("invalid object end")
			}
		case '[':
			if depth != 1 {
				return fmt.Errorf("unexpected array")
			}
			count := 0
			for d.More() {
				count++
				if count > MaxFiles {
					return fmt.Errorf("too many files")
				}
				if err := checkJSON(d, depth+1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return fmt.Errorf("invalid array end")
			}
		default:
			return fmt.Errorf("unexpected delimiter")
		}
	case string:
		if !ascii(v, 2048) {
			return fmt.Errorf("invalid JSON string")
		}
	case json.Number:
		// All v1 numeric fields are nonnegative. Reject -0 as well: JSON
		// libraries differ in whether they decode it as an integer or float.
		if strings.HasPrefix(v.String(), "-") {
			return fmt.Errorf("negative JSON number")
		}
	default:
		return fmt.Errorf("null and boolean values are not permitted")
	}
	return nil
}

func ascii(s string, max int) bool {
	if len(s) == 0 || len(s) > max {
		return false
	}
	for i := range s {
		if s[i] < 32 || s[i] > 126 {
			return false
		}
	}
	return true
}

func identifier(s string) bool {
	if len(s) == 0 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return s[0] != '-' && s[len(s)-1] != '-'
}

func validPath(s string) bool {
	if len(s) == 0 || len(s) > 240 {
		return false
	}
	parts := strings.Split(s, "/")
	if len(parts) > 8 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 64 || p[0] == '.' || p[len(p)-1] == '.' {
			return false
		}
		for _, c := range p {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
				return false
			}
		}
		stem := strings.SplitN(p, ".", 2)[0]
		if stem == "con" || stem == "prn" || stem == "aux" || stem == "nul" || len(stem) == 4 && (strings.HasPrefix(stem, "com") || strings.HasPrefix(stem, "lpt")) && stem[3] >= '0' && stem[3] <= '9' {
			return false
		}
	}
	return s != "manifest.json" && s != "manifest.sig"
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != 1 || !identifier(m.AreaID) || m.Release == 0 || m.Release > 1<<53-1 {
		return fmt.Errorf("invalid manifest identity/version")
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", m.CreatedAt)
	if err != nil || t.Format("2006-01-02T15:04:05Z") != m.CreatedAt {
		return fmt.Errorf("created_at must be UTC seconds")
	}
	if len(m.Files) == 0 || len(m.Files) > MaxFiles {
		return fmt.Errorf("file count outside limit")
	}
	paths := make(map[string]bool, len(m.Files))
	var total int64
	for i, f := range m.Files {
		if !validPath(f.Path) || i > 0 && m.Files[i-1].Path >= f.Path {
			return fmt.Errorf("paths must be portable, unique and sorted: %q", f.Path)
		}
		paths[f.Path] = true
		if f.Size < 0 || f.Size > MaxFileBytes || f.Size > MaxTotalBytes-total {
			return fmt.Errorf("file size outside limit: %s", f.Path)
		}
		total += f.Size
		hash, err := hex.DecodeString(f.SHA256)
		if err != nil || len(hash) != 32 || strings.ToLower(f.SHA256) != f.SHA256 {
			return fmt.Errorf("invalid SHA-256: %s", f.Path)
		}
		if !ascii(f.Format, 64) || !ascii(f.Source, 2048) || !ascii(f.License, 256) {
			return fmt.Errorf("invalid file metadata: %s", f.Path)
		}
	}
	for _, f := range m.Files {
		parts := strings.Split(f.Path, "/")
		for i := 1; i < len(parts); i++ {
			if paths[strings.Join(parts[:i], "/")] {
				return fmt.Errorf("file/directory collision: %s", f.Path)
			}
		}
	}
	return nil
}

// Sign uses a 32-byte Ed25519 seed. Keep production seeds outside pack/repo data.
func Sign(m Manifest, seed []byte) ([]byte, []byte, error) {
	if len(seed) != ed25519.SeedSize {
		return nil, nil, fmt.Errorf("invalid signing seed length")
	}
	if err := m.Validate(); err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > MaxManifestBytes {
		return nil, nil, fmt.Errorf("manifest exceeds size limit")
	}
	sig := ed25519.Sign(ed25519.NewKeyFromSeed(seed), append([]byte(Domain), raw...))
	return raw, sig, nil
}

// Verify checks a caller-trusted key before parsing signed bytes. A public key
// supplied by the pack itself must never be treated as trusted.
func Verify(raw, sig []byte, key ed25519.PublicKey) (Manifest, error) {
	if len(raw) == 0 || len(raw) > MaxManifestBytes || len(sig) != ed25519.SignatureSize || len(key) != ed25519.PublicKeySize {
		return Manifest{}, fmt.Errorf("invalid manifest/signature/key length")
	}
	if !ed25519.Verify(key, append([]byte(Domain), raw...), sig) {
		return Manifest{}, fmt.Errorf("invalid manifest signature")
	}
	return Parse(raw)
}
