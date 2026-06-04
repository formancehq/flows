package v1

import (
	"net/http"

	bunpaginate "github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"

	api2 "github.com/formancehq/orchestration/internal/api"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
)

func listWorkflows(backend api2.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		workflows, err := backend.ListWorkflows(r.Context(), bunpaginate.OffsetPaginatedQuery[any]{})
		if err != nil {
			api.InternalServerError(w, r, err)
			return
		}

		api.Ok(w, workflows.Data)
	}
}
