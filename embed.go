package odem

import (
	"embed"
	"io"
)

//go:embed README.md
//go:embed LICENSE
var files embed.FS

func GetFile(path string) (io.ReadCloser, error) { return files.Open(path) }
