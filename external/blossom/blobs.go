package blossom

const XContentLength = "X-Content-Length"
const XContentType = "X-Content-Length"
const XUploadMessage = "X-Upload-Message"
const XSHA256 = "X-SHA-256"

type Blob struct {
	Data []byte
	Size uint64
	Name string
	Type string
}

type DBBlobData struct {
	Path      string
	Sha256    []byte
	CreatedAt uint64 `db:"created_at"`
	Data      Blob
	Pubkey    string
}

type BlobDescriptor struct {
	Url      string `json:"url"`
	Sha256   string `json:"sha256"`
	Size     uint64 `json:"size"`
	Type     string `json:"type"`
	Uploaded string `json:"uploaded"`
}
