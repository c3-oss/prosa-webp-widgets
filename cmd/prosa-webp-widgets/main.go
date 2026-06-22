// Command prosa-webp-widgets renders GitHub-profile WebP widgets from prosa analytics.
package main

import (
	"os"

	"github.com/c3-oss/prosa-webp-widgets/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
