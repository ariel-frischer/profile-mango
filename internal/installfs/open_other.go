//go:build !linux

package installfs

import "os"

func openReadNoFollow(path string) (*os.File, error) { return os.Open(path) }
