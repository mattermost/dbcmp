package testlib

import "embed"

//go:embed sql
//go:embed text
var assets embed.FS

func Assets() embed.FS {
	return assets
}
