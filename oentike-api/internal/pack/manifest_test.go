package pack

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample() Manifest {
	sum := sha256.Sum256([]byte("hello\n"))
	return Manifest{1, "nadl-05-31", 1, "2026-10-02T12:00:00Z", []File{{"data/snapshot.json", 6, hex.EncodeToString(sum[:]), "test-text/1", "synthetic:test", "CC0-1.0"}}}
}

func signed(t *testing.T, m Manifest) ([]byte, []byte, ed25519.PublicKey) {
	t.Helper()
	seed := make([]byte, 32) // Public test-only seed. Never a production trust root.
	raw, sig, err := Sign(m, seed)
	if err != nil {
		t.Fatal(err)
	}
	return raw, sig, ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
}

func TestSignatureBinding(t *testing.T) {
	raw, sig, key := signed(t, sample())
	if _, err := Verify(raw, sig, key); err != nil {
		t.Fatal(err)
	}
	for name, check := range map[string]func() error{
		"changed manifest": func() error {
			_, err := Verify(bytes.Replace(raw, []byte("nadl-05-31"), []byte("nadl-05-32"), 1), sig, key)
			return err
		},
		"whitespace":        func() error { _, err := Verify(append(bytes.Clone(raw), '\n'), sig, key); return err },
		"changed signature": func() error { s := bytes.Clone(sig); s[0] ^= 1; _, err := Verify(raw, s, key); return err },
		"wrong key":         func() error { k := bytes.Clone(key); k[0] ^= 1; _, err := Verify(raw, sig, k); return err },
		"short key":         func() error { _, err := Verify(raw, sig, key[:31]); return err },
		"short signature":   func() error { _, err := Verify(raw, sig[:63], key); return err },
		"long signature":    func() error { _, err := Verify(raw, append(bytes.Clone(sig), 0), key); return err },
		"no domain": func() error {
			s := ed25519.Sign(ed25519.NewKeyFromSeed(make([]byte, 32)), raw)
			_, err := Verify(raw, s, key)
			return err
		},
		"oversized": func() error { _, err := Verify(make([]byte, MaxManifestBytes+1), sig, key); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := check(); err == nil {
				t.Fatal("accepted invalid signature input")
			}
		})
	}
	if _, _, err := Sign(sample(), nil); err == nil {
		t.Fatal("accepted invalid seed")
	}
}

func TestSignedInvalidJSON(t *testing.T) {
	raw, _, key := signed(t, sample())
	base := string(raw)
	cases := map[string]string{
		"duplicate":         strings.Replace(base, `"release":1`, `"release":1,"release":2`, 1),
		"escaped duplicate": strings.Replace(base, `"release":1`, `"release":1,"rele\u0061se":2`, 1),
		"nested duplicate":  strings.Replace(base, `"size":6`, `"size":6,"size":7`, 1),
		"case alias":        strings.Replace(base, `"release"`, `"Release"`, 1),
		"missing size":      strings.Replace(base, `"size":6,`, "", 1),
		"missing source":    strings.Replace(base, `"source":"synthetic:test",`, "", 1),
		"unknown":           strings.Replace(base, `"release":1`, `"release":1,"extra":0`, 1),
		"null":              strings.Replace(base, `"size":6`, `"size":null`, 1),
		"float":             strings.Replace(base, `"size":6`, `"size":6.0`, 1),
		"exponent":          strings.Replace(base, `"size":6`, `"size":6e0`, 1),
		"negative":          strings.Replace(base, `"size":6`, `"size":-1`, 1),
		"negative zero":     strings.Replace(base, `"size":6`, `"size":-0`, 1),
		"overflow":          strings.Replace(base, `"release":1`, `"release":18446744073709551616`, 1),
		"unsafe integer":    strings.Replace(base, `"release":1`, `"release":9007199254740992`, 1),
		"version":           strings.Replace(base, `"schema_version":1`, `"schema_version":2`, 1),
		"bad surrogate":     strings.Replace(base, "synthetic:test", `\ud800`, 1),
		"invalid utf8":      strings.Replace(base, "synthetic:test", "\xff", 1),
		"trailing":          base + "{}",
		"deep":              `{"files":` + strings.Repeat("[", 100) + strings.Repeat("]", 100) + `}`,
		"path traversal":    strings.Replace(base, "data/snapshot.json", "../snapshot.json", 1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			sig := ed25519.Sign(ed25519.NewKeyFromSeed(make([]byte, 32)), append([]byte(Domain), []byte(input)...))
			if _, err := Verify([]byte(input), sig, key); err == nil {
				t.Fatal("accepted signed invalid JSON")
			}
		})
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(ed25519.NewKeyFromSeed(make([]byte, 32)), append([]byte(Domain), pretty.Bytes()...))
	if _, err := Verify(pretty.Bytes(), sig, key); err != nil {
		t.Fatalf("signed whitespace is allowed: %v", err)
	}
}

func TestManifestConstraints(t *testing.T) {
	for _, path := range []string{"../a", "/a", "a//b", "a/./b", `a\b`, "A.json", "a.", "a/../b", "nul.txt", "con", "com1.txt", "lpt9", "a:b", "a b", "manifest.json", "manifest.sig", ".hidden", strings.Repeat("a/", 9) + "b", strings.Repeat("a", 65)} {
		t.Run(path, func(t *testing.T) {
			m := sample()
			m.Files[0].Path = path
			if err := m.Validate(); err == nil {
				t.Fatal("accepted unsafe path")
			}
		})
	}
	for name, mutate := range map[string]func(*Manifest){
		"empty":     func(m *Manifest) { m.Files = nil },
		"duplicate": func(m *Manifest) { m.Files = append(m.Files, m.Files[0]) },
		"prefix":    func(m *Manifest) { f := m.Files[0]; f.Path = "data"; m.Files = append([]File{f}, m.Files...) },
		"unsorted":  func(m *Manifest) { f := m.Files[0]; f.Path = "a"; m.Files = append(m.Files, f) },
		"too large": func(m *Manifest) { m.Files[0].Size = MaxFileBytes + 1 },
		"total": func(m *Manifest) {
			for i := 0; i < 5; i++ {
				f := m.Files[0]
				f.Path = fmt.Sprintf("z%d", i)
				f.Size = MaxFileBytes
				m.Files = append(m.Files, f)
			}
		},
		"too many":       func(m *Manifest) { m.Files = make([]File, MaxFiles+1) },
		"hash":           func(m *Manifest) { m.Files[0].SHA256 = strings.Repeat("g", 64) },
		"uppercase hash": func(m *Manifest) { m.Files[0].SHA256 = strings.ToUpper(m.Files[0].SHA256) },
		"time offset":    func(m *Manifest) { m.CreatedAt = "2026-10-02T12:00:00+00:00" },
		"time precision": func(m *Manifest) { m.CreatedAt = "2026-10-02T12:00:00.1Z" },
		"area":           func(m *Manifest) { m.AreaID = "../bad" },
	} {
		t.Run(name, func(t *testing.T) {
			m := sample()
			mutate(&m)
			if err := m.Validate(); err == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
}

func payload(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data/snapshot.json"), []byte("hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPayloadIntegrity(t *testing.T) {
	for _, scenario := range []string{"valid", "changed", "truncated", "missing", "extra", "extra directory", "symlink", "parent symlink", "empty"} {
		t.Run(scenario, func(t *testing.T) {
			root := payload(t)
			m, err := Build(root, sample())
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "data/snapshot.json")
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			switch scenario {
			case "changed":
				must(os.WriteFile(path, []byte("HELLO\n"), 0600))
			case "truncated":
				must(os.Truncate(path, 1))
			case "missing":
				must(os.Remove(path))
			case "extra":
				must(os.WriteFile(filepath.Join(root, "extra"), nil, 0600))
			case "extra directory":
				must(os.Mkdir(filepath.Join(root, "extra"), 0700))
			case "symlink":
				must(os.Remove(path))
				must(os.Symlink(filepath.Join(t.TempDir(), "outside"), path))
			case "parent symlink":
				must(os.Remove(path))
				must(os.Remove(filepath.Dir(path)))
				must(os.Symlink(t.TempDir(), filepath.Dir(path)))
			case "empty":
				must(os.Truncate(path, 0))
				m, err = Build(root, sample())
				must(err)
			}
			err = VerifyFiles(root, m)
			if scenario == "valid" || scenario == "empty" {
				must(err)
			} else if err == nil {
				t.Fatal("accepted invalid payload")
			}
		})
	}
}

func TestBuildDoesNotMutateTemplate(t *testing.T) {
	m := sample()
	m.Files[0].Size = 0
	m.Files[0].SHA256 = strings.Repeat("0", 64)
	got, err := Build(payload(t), m)
	if err != nil {
		t.Fatal(err)
	}
	if m.Files[0].Size != 0 || got.Files[0].Size != 6 || got.Files[0].SHA256 != sample().Files[0].SHA256 {
		t.Fatal("incorrect build or mutated caller template")
	}
}

func TestGoldenVector(t *testing.T) {
	raw, sig, key := signed(t, sample())
	b, err := os.ReadFile("testdata/vector.json")
	if err != nil {
		t.Fatalf("vector missing; manifest=%s signature=%x public=%x", raw, sig, key)
	}
	var v struct {
		SeedHex      string `json:"seed_hex"`
		PublicKeyHex string `json:"public_key_hex"`
		Manifest     string `json:"manifest"`
		SignatureHex string `json:"signature_hex"`
		PayloadHex   string `json:"payload_hex"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if string(raw) != v.Manifest || hex.EncodeToString(sig) != v.SignatureHex || hex.EncodeToString(key) != v.PublicKeyHex || v.SeedHex != strings.Repeat("0", 64) {
		t.Fatal("wire contract differs from frozen vector")
	}
	data, err := hex.DecodeString(v.PayloadHex)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != sample().Files[0].SHA256 {
		t.Fatal("vector payload hash mismatch")
	}
}

func FuzzParse(f *testing.F) {
	raw, _ := json.Marshal(sample())
	f.Add(raw)
	f.Add([]byte(`{"files":null}`))
	f.Add([]byte(`{"release":1,"release":2}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := Parse(data)
		if err == nil {
			if err := m.Validate(); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func BenchmarkVerifyManifest(b *testing.B) {
	raw, sig, err := Sign(sample(), make([]byte, 32))
	if err != nil {
		b.Fatal(err)
	}
	key := ed25519.NewKeyFromSeed(make([]byte, 32)).Public().(ed25519.PublicKey)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := Verify(raw, sig, key); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyFiles16MiB(b *testing.B) {
	root := b.TempDir()
	f, err := os.Create(filepath.Join(root, "data.bin"))
	if err != nil {
		b.Fatal(err)
	}
	chunk := make([]byte, 64<<10)
	for i := 0; i < 256; i++ {
		if _, err := f.Write(chunk); err != nil {
			b.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		b.Fatal(err)
	}
	m := sample()
	m.Files[0].Path = "data.bin"
	m, err = Build(root, m)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(16 << 20)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := VerifyFiles(root, m); err != nil {
			b.Fatal(err)
		}
	}
}
