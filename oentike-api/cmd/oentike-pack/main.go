// oentike-pack generates detached signed manifests and verifies payload trees.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"oentike-api/internal/pack"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "oentike-pack:", err)
		os.Exit(1)
	}
}

func readLimited(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("invalid input file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("input exceeds limit: %s", path)
	}
	return b, nil
}

func readKey(path string) ([]byte, error) {
	b, err := readLimited(path, 66)
	if err != nil {
		return nil, err
	}
	key, err := hex.DecodeString(strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r"))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("key file must contain 64 hexadecimal digits")
	}
	return key, nil
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "sign" && args[0] != "verify" {
		return fmt.Errorf("usage: oentike-pack sign|verify -root PAYLOAD [flags]; use -h for flags")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(out)
	root := flags.String("root", "", "immutable payload directory (required)")
	manifest := flags.String("manifest", "", "manifest JSON; for sign, sizes and hashes are recomputed (required)")
	var seed, dest, signature, public *string
	if args[0] == "sign" {
		seed = flags.String("seed-file", "", "external Ed25519 seed file, 64 hex digits (required)")
		dest = flags.String("out", "", "new directory for manifest.json and manifest.sig (required)")
	} else {
		signature = flags.String("signature", "", "detached raw 64-byte signature (required)")
		public = flags.String("public-key-file", "", "caller-trusted public key, 64 hex digits (required)")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *root == "" || *manifest == "" {
		return fmt.Errorf("root and manifest are required; positional arguments are not supported")
	}
	raw, err := readLimited(*manifest, pack.MaxManifestBytes)
	if err != nil {
		return err
	}
	if args[0] == "verify" {
		if *signature == "" || *public == "" {
			return fmt.Errorf("signature and public-key-file are required")
		}
		key, err := readKey(*public)
		if err != nil {
			return err
		}
		sig, err := readLimited(*signature, ed25519.SignatureSize)
		if err != nil {
			return err
		}
		m, err := pack.Verify(raw, sig, key)
		if err != nil {
			return err
		}
		if err := pack.VerifyFiles(*root, m); err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Verified %s release %d (%d files); format contents and activation not checked\n", m.AreaID, m.Release, len(m.Files))
		return err
	}
	if *seed == "" || *dest == "" {
		return fmt.Errorf("seed-file and out are required")
	}
	key, err := readKey(*seed)
	if err != nil {
		return err
	}
	template, err := pack.Parse(raw)
	if err != nil {
		return err
	}
	m, err := pack.Build(*root, template)
	if err != nil {
		return err
	}
	raw, sig, err := pack.Sign(m, key)
	if err != nil {
		return err
	}
	// Metadata output is deliberately separate from the immutable payload tree.
	// Refuse existing directories, so a previous release is never overwritten.
	rootAbs, err := filepath.EvalSymlinks(*root)
	if err != nil {
		return err
	}
	rootAbs, err = filepath.Abs(rootAbs)
	if err != nil {
		return err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(*dest))
	if err != nil {
		return err
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		return err
	}
	target := filepath.Join(parent, filepath.Base(*dest))
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return err
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("out must be outside the payload directory")
	}
	if err := os.Mkdir(target, 0700); err != nil {
		return err
	}
	for _, f := range []struct {
		name string
		data []byte
	}{{"manifest.json", raw}, {"manifest.sig", sig}} {
		path := filepath.Join(target, f.name)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("incomplete output at %s: %w", target, err)
		}
		_, writeErr := file.Write(f.data)
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("incomplete output at %s: %w", target, writeErr)
		}
		if closeErr != nil {
			return fmt.Errorf("incomplete output at %s: %w", target, closeErr)
		}
	}
	_, err = fmt.Fprintf(out, "Signed %s release %d (%d files): %s\n", m.AreaID, m.Release, len(m.Files), target)
	return err
}
