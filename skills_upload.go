package flashduty

import (
	"context"
	"fmt"
	"io"
	"strconv"
)

// SkillUploadRequest carries the skill archive and options for WriteUpload. It
// is hand-written rather than generated because the endpoint consumes
// multipart/form-data, which the generated JSON request path cannot encode.
type SkillUploadRequest struct {
	// File is the skill archive (.skill / .zip / .tar.gz / .tgz). Required.
	// Max 100 MB; oversized files are rejected before the body is read.
	File io.Reader
	// Filename is the multipart file name; defaults to "skill.zip".
	Filename string
	// TeamID scopes the created/upserted skill: 0 (default) = account-wide.
	// Ignored when replacing a specific skill via SkillID.
	TeamID int64
	// Replace overwrites an existing skill instead of failing on a name
	// collision — matched by SkillID if provided, otherwise by skill name.
	Replace bool
	// SkillID targets an existing skill when replacing (requires Replace).
	SkillID string
}

// WriteUpload uploads a skill archive to create or replace a skill.
//
// API: POST /safari/skill/upload (skill-write-upload).
func (s *SkillsService) WriteUpload(ctx context.Context, req *SkillUploadRequest) (*SkillItem, *Response, error) {
	if req == nil || req.File == nil {
		return nil, nil, fmt.Errorf("flashduty: SkillUploadRequest with a non-nil File is required")
	}
	filename := req.Filename
	if filename == "" {
		filename = "skill.zip"
	}
	fields := map[string]string{}
	if req.TeamID != 0 {
		fields["team_id"] = strconv.FormatInt(req.TeamID, 10)
	}
	if req.Replace {
		fields["replace"] = "true"
	}
	if req.SkillID != "" {
		fields["skill_id"] = req.SkillID
	}
	out := new(SkillItem)
	resp, err := s.client.uploadFile(ctx, "/safari/skill/upload", nil, fields, filename, req.File, out)
	if err != nil {
		return nil, resp, err
	}
	return out, resp, nil
}
