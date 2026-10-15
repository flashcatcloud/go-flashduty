package flashduty

import "context"

// List lists monitor folders. It preserves the original SDK method name.
func (s *RuleFoldersService) List(ctx context.Context) (*[]FolderItem, *Response, error) {
	return s.ReadList(ctx)
}
