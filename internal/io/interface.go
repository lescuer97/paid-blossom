package io

import "io"

type BlossomServerFS struct {
	w io.Writer
	r io.Reader
}

// func (blm BlossomServerFS) WriteBlob(path string, blob []byte) error {
//
// }

type BlossomIO interface {
	WriteBlob(path string, blob []byte) error
	// GetBlob(path string) ([]byte, error)
	// RemoveBlob(path string) error
}
