package database

import "ratasker/external/blossom"

type Database interface {
	AddBlob(data blossom.DBBlobData) error
	GetBlob(hash []byte) (blossom.DBBlobData, error)
	GetBlobLength(hash []byte) (uint64, error)
	// RemoveBlob(data blossom.StoredData) error
}
