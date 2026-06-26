package v1

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"

	api2 "github.com/formancehq/orchestration/internal/api"

	"github.com/formancehq/go-libs/v3/api"
)

func listWorkflows(backend api2.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Bound the query: without a page size, bunpaginate applies no LIMIT,
		// loading the entire workflows table per request.
		pageSize, err := bunpaginate.GetPageSize(r)
		if err != nil {
			api.BadRequest(w, "VALIDATION", err)
			return
		}

		workflows, err := backend.ListWorkflows(r.Context(), bunpaginate.OffsetPaginatedQuery[any]{
			PageSize: pageSize,
		})
		if err != nil {
			api.InternalServerError(w, r, err)
			return
		}

		api.Ok(w, workflows.Data)
	}
}
