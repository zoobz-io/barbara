// Command adminspec dumps the admin API's OpenAPI specification to disk. No
// server, no database — the output feeds the @barbara/admin-sdk client
// generator in the web monorepo. See internal/openapispec for how the spec is
// produced.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/zoobz-io/barbara/admin/handlers"
	"github.com/zoobz-io/barbara/internal/openapispec"
)

// defaultOut is where the spec lands when no path argument is given: the admin
// SDK package's committed spec snapshot.
const defaultOut = "web/packages/admin-sdk/data/openapi.json"

func main() {
	out := defaultOut
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := openapispec.Dump(handlers.ConfigureOpenAPI, handlers.All(), out); err != nil {
		log.Fatalf("adminspec: %v", err)
	}
	fmt.Printf("wrote %s\n", out)
}
