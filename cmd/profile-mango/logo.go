package main

import (
	"fmt"
	"io"
)

const readmeLogo = `█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█`

func writeInitLogo(writer io.Writer) error {
	styles := stylesFor(writer, true)
	if _, err := fmt.Fprintln(writer, styles.heading(readmeLogo)); err != nil {
		return fmt.Errorf("writing init logo: %w", err)
	}
	return nil
}
