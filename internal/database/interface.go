package database

import "ratasker/external/blossom"

type Database interface {
	AddBlob(data blossom.DBBlobData) error
	GetBlob(hash []byte) (blossom.DBBlobData, error)
	// RemoveBlob(data blossom.StoredData) error
}
