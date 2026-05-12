package file

import "time"

type File struct {
	ID        string     `json:"id"`
	Filename  string     `json:"filename"`
	Path      string     `json:"path"`
	Size      int64      `json:"size"`
	MimeType  string     `json:"mime_type"`
	Category  string     `json:"category"`
	Format    string     `json:"format"`
	Storage   string     `json:"storage"`
	Trashed   bool       `json:"trashed"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type UploadResult struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	Format    string `json:"format"`
	Category  string `json:"category"`
}

type TrashItem struct {
	File       File   `json:"file"`
	OriginalPath string `json:"original_path"`
}
