package main

import "embed"

// use embed
var _ = (*embed.FS)(nil)

//go:embed resource/shadow.png
var icon []byte
