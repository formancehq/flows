package v1

import (
	"net/http"

	bunpaginate "github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
)

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
