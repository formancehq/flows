package v1

import (
	"net/http"

	bunpaginate "github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
)

// normalizePageSize applies the same bounds to decoded cursors as GetPageSize
// applies to explicit query parameters. Cursors are client-controlled base64
// JSON, so their embedded page size cannot be trusted.
func normalizePageSize(pageSize uint64) uint64 {
	if pageSize == 0 {
		return bunpaginate.QueryDefaultPageSize
	}
	if pageSize > bunpaginate.MaxPageSize {
		return bunpaginate.MaxPageSize
	}
	return pageSize
}

// renderCursor preserves the v1 {"data": [...]} response shape while adding
// enough metadata for clients to detect and retrieve subsequent pages.
func renderCursor[T any](w http.ResponseWriter, cursor bunpaginate.Cursor[T]) {
	sharedapi.RawOk(w, struct {
		Data     []T    `json:"data"`
		PageSize int    `json:"pageSize"`
		HasMore  bool   `json:"hasMore"`
		Previous string `json:"previous,omitempty"`
		Next     string `json:"next,omitempty"`
	}{
		Data:     cursor.Data,
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
	})
}
