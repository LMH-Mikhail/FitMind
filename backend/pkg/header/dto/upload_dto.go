package dto

type UploadFileResponse struct {
	OriginalName string `json:"originalName"`
	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType"`
	Size         int64  `json:"size"`
	RelativePath string `json:"relativePath"`
	ImageURL     string `json:"imageUrl"`
	PublicURL    string `json:"publicUrl"`
}
