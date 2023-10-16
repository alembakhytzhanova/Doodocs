package info

type ArchiveInfo struct {
	Filename    string
	ArchiveSize float64
	TotalSize   float64
	TotalFiles  int
	Files       []FileInfo
}

type FileInfo struct {
	FilePath string
	Size     float64
	MimeType string
}
