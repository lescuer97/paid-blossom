package blossom

type Blob struct {
	Data []byte
	Size uint64
	Name string
}

type DBBlobData struct {
	Path      string
	Sha256    []byte
	CreatedAt uint64 `db:"created_at"`
	Data      Blob
}

type BlobDescriptor struct {
	Url      string
	Sha256   string
	Size     uint64
	Type     string
	Uploaded uint64
}
