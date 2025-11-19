package models

type Choice struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type SearchResponse struct {
	RequestID string   `json:"request_id"`
	Choices   []string `json:"choices"`
}

type DownloadRequest struct {
	RequestID string   `json:"request_id"`
	Choices   []string `json:"choices"`
}
