// Package version carries the build version, injected at build time:
//
//	go build -ldflags "-X github.com/zhanhd/gitra/internal/version.Version=v1.2.3"
package version

// Version is "dev" for local builds and the release tag in published builds.
var Version = "dev"
