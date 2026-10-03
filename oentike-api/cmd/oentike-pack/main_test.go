package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "payload")
	if err := os.MkdirAll(filepath.Join(root, "data"), 0700); err != nil {
		t.Fatal(err)
	}
	vector, err := os.ReadFile("../../internal/pack/testdata/vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Manifest     string `json:"manifest"`
		SignatureHex string `json:"signature_hex"`
		PublicKeyHex string `json:"public_key_hex"`
	}
	if err := json.Unmarshal(vector, &v); err != nil {
		t.Fatal(err)
	}
	put := func(name string, data []byte) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	put("payload/data/snapshot.json", []byte("hello\n"))
	template := put("template.json", []byte(v.Manifest))
	seed := put("seed.hex", []byte(strings.Repeat("0", 64)+"\n"))
	public := put("public.hex", []byte(v.PublicKeyHex+"\n"))
	outDir := filepath.Join(dir, "signed")
	signArgs := []string{"sign", "-root", root, "-manifest", template, "-seed-file", seed, "-out", outDir}
	var output bytes.Buffer
	if err := run(signArgs, &output); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	sig, err := os.ReadFile(filepath.Join(outDir, "manifest.sig"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != v.Manifest || hex.EncodeToString(sig) != v.SignatureHex {
		t.Fatal("CLI output differs from vector")
	}
	verifyArgs := []string{"verify", "-root", root, "-manifest", filepath.Join(outDir, "manifest.json"), "-signature", filepath.Join(outDir, "manifest.sig"), "-public-key-file", public}
	if err := run(verifyArgs, &output); err != nil {
		t.Fatal(err)
	}
	if err := run(signArgs, &output); err == nil {
		t.Fatal("overwrote existing output")
	}
	for _, badDest := range []string{filepath.Join(root, "signed"), root} {
		args := append([]string(nil), signArgs...)
		args[len(args)-1] = badDest
		if err := run(args, &output); err == nil {
			t.Fatal("wrote metadata into payload")
		}
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	args := append([]string(nil), signArgs...)
	args[len(args)-1] = filepath.Join(alias, "signed")
	if err := run(args, &output); err == nil {
		t.Fatal("wrote through symlink into payload")
	}
	wrongKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, 32)).Public().(ed25519.PublicKey)
	put("public.hex", []byte(hex.EncodeToString(wrongKey)))
	if err := run(verifyArgs, &output); err == nil {
		t.Fatal("accepted wrong trust root")
	}
	put("public.hex", []byte(v.PublicKeyHex))
	put("payload/data/snapshot.json", []byte("HELLO\n"))
	if err := run(verifyArgs, &output); err == nil {
		t.Fatal("accepted modified payload")
	}
	for _, args := range [][]string{nil, {"unknown"}, {"verify"}, {"sign"}, {"verify", "-root", root, "-manifest", template, "unexpected"}} {
		if err := run(args, &output); err == nil {
			t.Fatalf("accepted missing/invalid flags: %v", args)
		}
	}
}

func TestReadLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, bytes.Repeat([]byte{'a'}, 67), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readKey(path); err == nil {
		t.Fatal("accepted oversized key")
	}
	if _, err := readLimited(path, 64); err == nil {
		t.Fatal("accepted oversized input")
	}
}
