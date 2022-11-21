package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var embedFiles embed.FS

func GetFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embedFiles, "ui")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}
