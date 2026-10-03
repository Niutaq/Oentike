//go:build darwin || linux

package pack

import (
	"crypto/sha256"
	"fmt"
	"os"
	"syscall"
)

func hashFile(root *os.Root, path string, expected int64) (int64, [32]byte, error) {
	var zero [32]byte
	f, err := root.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return 0, zero, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, zero, err
	}
	size := info.Size()
	if !info.Mode().IsRegular() || size < 0 || size > MaxFileBytes || uint64(size) > uint64(^uint(0)>>1) || expected >= 0 && size != expected {
		return 0, zero, fmt.Errorf("invalid file type or size: %s", path)
	}
	if size == 0 {
		return 0, sha256.Sum256(nil), nil
	}
	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return 0, zero, err
	}
	hash := sha256.Sum256(data)
	if err := syscall.Munmap(data); err != nil {
		return 0, zero, err
	}
	return size, hash, nil
}
