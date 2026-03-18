package main

import (
	"os"

	"github.com/heodongun/freeapi/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
