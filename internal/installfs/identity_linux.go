//go:build linux

package installfs

import (
	"os"
	"syscall"
)

func fileIdentity(info os.FileInfo) Identity {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return Identity{Size: info.Size(), ModTime: info.ModTime().UnixNano(), Mode: uint32(info.Mode().Perm())}
	}
	return Identity{Device: uint64(stat.Dev), Inode: uint64(stat.Ino), Size: info.Size(), ModTime: info.ModTime().UnixNano(), Mode: uint32(info.Mode().Perm())}
}

func linkCount(info os.FileInfo) uint64 {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 1
	}
	return uint64(stat.Nlink)
}

func sameIdentity(left, right os.FileInfo) bool { return fileIdentity(left) == fileIdentity(right) }
