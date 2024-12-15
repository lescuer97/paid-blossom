package io

type BlossomIO interface {
	WriteBlob(filename string, blob []byte) error
	GetBlob(path string) ([]byte, error)
	RemoveBlob(path string) error
	GetStoragePath() string
}
