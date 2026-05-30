package flashduty

// ListOptions holds the pagination inputs shared by every list endpoint. Embed
// it in a service's request struct; zero values are omitted so they never
// override the server's defaults (the backend uses p=1, limit=20).
type ListOptions struct {
	// Page is the 1-based page number (wire field "p").
	Page int `json:"p,omitempty"`
	// Limit caps the number of items returned per page.
	Limit int `json:"limit,omitempty"`
	// SearchAfterCtx is the opaque cursor echoed by the previous page for deep
	// pagination; pass it back to fetch the next page.
	SearchAfterCtx string `json:"search_after_ctx,omitempty"`
}
