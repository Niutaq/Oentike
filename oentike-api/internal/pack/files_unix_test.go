//go:build darwin || linux

package pack

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRejectFIFOAndOversizedFile(t *testing.T) {
	for _, fifo := range []bool{false, true} {
		root := payload(t)
		path := filepath.Join(root, "data/snapshot.json")
		if fifo {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0600); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.Truncate(path, MaxFileBytes+1); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Build(root, sample()); err == nil {
			t.Fatal("accepted invalid file")
		}
		if err := VerifyFiles(root, sample()); err == nil {
			t.Fatal("accepted invalid file")
		}
	}
}
