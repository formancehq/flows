package v2

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"

	"github.com/formancehq/orchestration/internal/api"

	sharedapi "github.com/formancehq/go-libs/v2/api"
)

func listWorkflows(backend api.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query, err := bunpaginate.Extract[bunpaginate.OffsetPaginatedQuery[any]](r, func() (*bunpaginate.OffsetPaginatedQuery[any], error) {
			pageSize, err := bunpaginate.GetPageSize(r)
			if err != nil {
				return nil, err
			}
			return &bunpaginate.OffsetPaginatedQuery[any]{
				PageSize: pageSize,
			}, nil
		})
		if err != nil {
			sharedapi.BadRequest(w, "VALIDATION", err)
			return
		}

		workflows, err := backend.ListWorkflows(r.Context(), *query)
		if err != nil {
			sharedapi.InternalServerError(w, r, err)
			return
		}

		sharedapi.RenderCursor(w, *workflows)
	}
}
