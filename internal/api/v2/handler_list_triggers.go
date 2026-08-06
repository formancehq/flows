package v2

import (
	"net/http"

	bunpaginate "github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/orchestration/internal/triggers"

	"github.com/formancehq/orchestration/internal/api"

	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
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

		sharedapi.RenderCursor(w, *triggers)
	}
}
