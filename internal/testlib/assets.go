package testlib

import "embed"

//go:embed sql
var assets embed.FS

func Assets() embed.FS {
	return assets
}
