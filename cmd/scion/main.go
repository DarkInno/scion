package main

import (
	"context"
	"fmt"
	"os"

	"github.com/DarkInno/scion/internal/bundle"
	"github.com/DarkInno/scion/internal/cli"
	"github.com/DarkInno/scion/internal/registry"
)

func main() {
	reg, err := registry.NewBundle(bundle.RegistryZip, bundle.ManifestJSON)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "scion: failed to load embedded registry: %v\n", err)
		os.Exit(1)
	}

	app := cli.New(reg)
	os.Exit(app.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
