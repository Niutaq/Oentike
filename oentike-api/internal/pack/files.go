package pack

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
)

// Build computes sizes and hashes, replacing those fields in the template.
// root must be a private, immutable staging directory throughout this call.
// The payload directory contains only listed files and their parent directories;
// manifest/signature are stored separately by the CLI.
func Build(root string, template Manifest) (Manifest, error) {
	m := template
	m.Files = append([]File(nil), template.Files...)
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	for i := range m.Files {
		m.Files[i].Size = 0
		m.Files[i].SHA256 = fmt.Sprintf("%064d", 0)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return Manifest{}, err
	}
	defer r.Close()
	if err := inventory(r, m); err != nil {
		return Manifest{}, err
	}
	for i := range m.Files {
		size, hash, err := hashFile(r, m.Files[i].Path, -1)
		if err != nil {
			return Manifest{}, err
		}
		m.Files[i].Size, m.Files[i].SHA256 = size, hex.EncodeToString(hash[:])
	}
	return m, m.Validate()
}

// VerifyFiles checks the complete payload inventory and hashes. Call Verify first
// to establish authenticity. Files must remain immutable during and after checking.
func VerifyFiles(root string, m Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := inventory(r, m); err != nil {
		return err
	}
	for _, f := range m.Files {
		_, hash, err := hashFile(r, f.Path, f.Size)
		if err != nil {
			return err
		}
		if hex.EncodeToString(hash[:]) != f.SHA256 {
			return fmt.Errorf("hash mismatch: %s", f.Path)
		}
	}
	return nil
}

func inventory(r *os.Root, m Manifest) error {
	var total int64
	want := make(map[string]bool, len(m.Files))
	dirs := map[string]bool{".": true}
	for _, f := range m.Files {
		want[f.Path] = true
		for i := range f.Path {
			if f.Path[i] == '/' {
				dirs[f.Path[:i]] = true
			}
		}
	}
	// ReadDir is batched so a malicious directory cannot allocate an unbounded
	// slice before we discover the first unlisted entry.
	var walk func(string) error
	walk = func(path string) error {
		d, err := r.Open(path)
		if err != nil {
			return err
		}
		defer d.Close()
		for {
			entries, readErr := d.ReadDir(64)
			for _, entry := range entries {
				name := entry.Name()
				if path != "." {
					name = path + "/" + name
				}
				if entry.Type()&fs.ModeSymlink != 0 {
					return fmt.Errorf("symlink forbidden: %s", name)
				}
				if entry.IsDir() {
					if !dirs[name] {
						return fmt.Errorf("unlisted directory: %s", name)
					}
					if err := walk(name); err != nil {
						return err
					}
				} else {
					if !want[name] || !entry.Type().IsRegular() {
						return fmt.Errorf("unlisted or special file: %s", name)
					}
					info, err := entry.Info()
					if err != nil {
						return err
					}
					if !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > MaxFileBytes || info.Size() > MaxTotalBytes-total {
						return fmt.Errorf("payload size outside limit: %s", name)
					}
					total += info.Size()
					delete(want, name)
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					break
				}
				return readErr
			}
		}
		return nil
	}
	if err := walk("."); err != nil {
		return err
	}
	if len(want) != 0 {
		return fmt.Errorf("missing payload files")
	}
	return nil
}
