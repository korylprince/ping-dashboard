package main

import (
	"embed"
	"fmt"
	"net/http"
	"os"
	"time"
)

//go:embed ui/dist
var dist embed.FS

// EmbedFS is a wrapper around an embed.FS that supports caching by using the modification time of the Go binary
type EmbedFS struct {
	http.FileSystem
}

// Open implements http.FileSystem
func (f *EmbedFS) Open(name string) (http.File, error) {
	file, err := f.FileSystem.Open(name)
	return &EmbedFile{File: file}, err
}

// EmbedFile wraps an http.File. See EmbedFS for more information
type EmbedFile struct {
	http.File
}

// Stat implements http.File
func (f *EmbedFile) Stat() (os.FileInfo, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat file: %w", err)
	}
	binInfo, err := os.Stat(os.Args[0])
	if err != nil {
		return nil, fmt.Errorf("could not get binary ModTime: %w", err)
	}
	return &EmbedFileInfo{FileInfo: fi, modTime: binInfo.ModTime()}, err
}

// EmbedFileInfo wraps an os.FileInfo. See EmbedFS for more information
type EmbedFileInfo struct {
	os.FileInfo
	modTime time.Time
}

// ModTime implements fs.FileInfo
func (f *EmbedFileInfo) ModTime() time.Time {
	return f.modTime
}
