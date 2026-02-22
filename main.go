package main

import (
	"os"

	"github.com/mebn/sl/internal/sl"
)

func main() {
	os.Exit(sl.Run(os.Args[1:], os.Stdout, os.Stderr))
}
