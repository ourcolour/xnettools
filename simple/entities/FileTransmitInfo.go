package entities

type FileTransmitInfo struct {
	FileName string
	FileSize int64

	BufferSize int64
	Data       []byte

	BatchNo      int64
	TotalBatches int64
}
