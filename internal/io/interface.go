package io

type BlossomIO interface {
	WriteBlob(blob []byte) error
	GetBlob(path string) ([]byte, error)
	RemoveBlob(path string) error
	GetStoragePath() string
}
