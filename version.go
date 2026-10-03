// Package dockerctl holds build metadata shared by the CLI and internal packages.
package dockerctl

// Version can be overridden at build time:
// go build -ldflags "-X github.com/TimCares/go-dockerctl.Version=$(git describe --tags)".
var Version = "v0.1.0"
