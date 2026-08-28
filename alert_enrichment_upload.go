package flashduty

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// MappingDataUploadRequest carries the CSV file and options for
// MappingDataWriteUpload. It is hand-written rather than generated because the
// endpoint consumes multipart/form-data, which the generated JSON request path
// cannot encode.
type MappingDataUploadRequest struct {
	// SchemaID is the ID of the target mapping schema (ObjectID hex). Required.
	SchemaID string
	// DoNotTruncateFirst appends the CSV rows to the existing data instead of
	// truncating it first.
	DoNotTruncateFirst bool
	// File is the CSV content. Required. Max 100 MB; the header row must
	// include all of the schema's source/result label names.
	File io.Reader
	// Filename is the multipart file name; defaults to "mapping.csv".
	Filename string
}

// MappingDataWriteUpload bulk-loads mapping data rows from a CSV file. By
// default the existing data is truncated before the new rows load; set
// DoNotTruncateFirst to append instead.
//
// API: POST /enrichment/mapping/data/upload (mapping-data-write-upload).
func (s *AlertEnrichmentService) MappingDataWriteUpload(ctx context.Context, req *MappingDataUploadRequest) (*Response, error) {
	if req == nil || req.File == nil {
		return nil, fmt.Errorf("flashduty: MappingDataUploadRequest with a non-nil File is required")
	}
	filename := req.Filename
	if filename == "" {
		filename = "mapping.csv"
	}
	query := url.Values{"schema_id": {req.SchemaID}}
	if req.DoNotTruncateFirst {
		query.Set("do_not_truncate_first", "TRUE")
	}
	return s.client.uploadFile(ctx, "/enrichment/mapping/data/upload", query, nil, filename, req.File, nil)
}
