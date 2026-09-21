//go:build !linux

package installfs

import "os"

func fileIdentity(info os.FileInfo) Identity {
	return Identity{Size: info.Size(), ModTime: info.ModTime().UnixNano(), Mode: uint32(info.Mode().Perm())}
}

func linkCount(info os.FileInfo) uint64 { return 1 }

func sameIdentity(left, right os.FileInfo) bool { return fileIdentity(left) == fileIdentity(right) }
