package main

import (
	"os"

	"github.com/infrasutra/fsl/cmd/fsl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
