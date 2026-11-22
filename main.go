// Terraform / OpenTofu provider for Staysa Cloud.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/gattoufirabi/terraform-provider-staysa/internal/provider"
)

// version is set at release time (goreleaser).
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with support for debuggers such as delve")
	flag.Parse()
	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/gattoufirabi/staysa",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
