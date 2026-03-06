package main

import (
	"os"

	"github.com/infrasutra/fsl/cmd/fluxcms/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
