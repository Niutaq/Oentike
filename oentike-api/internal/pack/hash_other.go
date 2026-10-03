//go:build !darwin && !linux

package pack

import (
	"fmt"
	"os"
)

func hashFile(_ *os.Root, _ string, _ int64) (int64, [32]byte, error) {
	return 0, [32]byte{}, fmt.Errorf("pack mmap hashing currently requires Linux or macOS")
}
