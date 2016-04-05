package xdiff

import (
	"os"
	"syscall"
)

const has_mmap = true

func map_file(file *os.File, offset int64, size int) ([]byte, error) {
	data, err := syscall.Mmap(int(file.Fd()), offset, size, syscall.PROT_READ, syscall.MAP_SHARED)
	return data, err
}

func unmap_file(data []byte) error {
	return syscall.Munmap(data)
}
