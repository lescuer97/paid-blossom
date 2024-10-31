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
	Url      string `json:"url"`
	Sha256   string `json:"sha256"`
	Size     uint64 `json:"size"`
	Type     string `json:"type"`
	Uploaded uint64 `json:"uploaded"`
}
