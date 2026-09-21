package main

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

const readmeLogo = `█▀█ █▀█ █▀█ █▀▀ █ █   █▀▀   ─   █▀▄▀█ ▄▀█ █▄ █ █▀▀ █▀█
█▀▀ █▀▄ █▄█ █▀  █ █▄▄ ██▄       █ ▀ █ █▀█ █ ▀█ █▄█ █▄█`

func writeInitLogo(writer io.Writer) error {
	logoStyle := color.New(color.FgYellow, color.Bold)
	if !color.NoColor {
		logoStyle.EnableColor()
	}
	if _, err := logoStyle.Fprintln(writer, readmeLogo); err != nil {
		return fmt.Errorf("writing init logo: %w", err)
	}
	return nil
}
