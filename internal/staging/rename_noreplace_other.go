//go:build !linux && !darwin && !windows

package staging

import "errors"

var errAtomicNoReplaceUnsupported = errors.New("atomic no-replace staging is unsupported on this platform")

func renameNoReplace(_, _ string) error {
	return errAtomicNoReplaceUnsupported
}
