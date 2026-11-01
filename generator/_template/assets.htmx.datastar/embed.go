package assets

import "embed"

//go:embed css/* fonts/* img/* js/*
var AssetsFS embed.FS
