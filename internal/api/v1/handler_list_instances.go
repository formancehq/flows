package v1

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/orchestration/internal/workflow"

	api "github.com/formancehq/orchestration/internal/api"

	sharedapi "github.com/formancehq/go-libs/v3/api"
)

func listInstances(backend api.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Bound the query: without a page size, bunpaginate applies no LIMIT,
		// loading the entire workflow_instances table per request.
		pageSize, err := bunpaginate.GetPageSize(r)
		if err != nil {
			sharedapi.BadRequest(w, "VALIDATION", err)
			return
		}

		runs, err := backend.ListInstances(r.Context(), workflow.ListInstancesQuery{
			PageSize: pageSize,
			Options: workflow.ListInstancesOptions{
				WorkflowID: r.URL.Query().Get("workflowID"),
				Running:    sharedapi.QueryParamBool(r, "running"),
			},
		})
		if err != nil {
			sharedapi.InternalServerError(w, r, err)
			return
		}
		sharedapi.Ok(w, runs.Data)
	}
}
