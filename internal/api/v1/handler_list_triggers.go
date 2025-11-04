package v1

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/orchestration/internal/triggers"

	"github.com/formancehq/orchestration/internal/api"

	sharedapi "github.com/formancehq/go-libs/v3/api"
)

func listTriggers(backend api.Backend) func(writer http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		query, err := bunpaginate.Extract[triggers.ListTriggersQuery](r, func() (*triggers.ListTriggersQuery, error) {
			pageSize, err := bunpaginate.GetPageSize(r)
			if err != nil {
				return nil, err
			}

			name := r.URL.Query().Get("name")

			return &triggers.ListTriggersQuery{
				PageSize: pageSize,
				Options:  triggers.ListTriggerParams{Name: name},
			}, nil
		})
		if err != nil {
			sharedapi.BadRequest(w, "VALIDATION", err)
			return
		}

		triggers, err := backend.ListTriggers(r.Context(), *query)
		if err != nil {
			sharedapi.InternalServerError(w, r, err)
			return
		}

		sharedapi.Ok(w, triggers.Data)
	}
}
