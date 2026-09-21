//go:build !linux && !darwin && !windows

package staging

import "os"

func renameNoReplace(source, destination string) error {
	return os.Rename(source, destination)
}
