package main

import (
	"os"

	"github.com/bssm-oss/Free-API/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
