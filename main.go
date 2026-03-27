package main

import (
	"context"
	"log"

	"github.com/PureStorage-OpenConnect/terraform-provider-purefb/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/purestorage/purefb",
	})
	if err != nil {
		log.Fatal(err)
	}
}
