package v1

import (
	"net/http"

	bunpaginate "github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/go-chi/chi/v5"

	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/triggers"
)

func listTriggersOccurrences(backend api.Backend) func(writer http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := bunpaginate.Extract[triggers.ListTriggersOccurrencesQuery](r, func() (*triggers.ListTriggersOccurrencesQuery, error) {
			pageSize, err := bunpaginate.GetPageSize(r)
			if err != nil {
				return nil, err
			}
			return &triggers.ListTriggersOccurrencesQuery{
				PageSize: pageSize,
				Options: triggers.ListTriggersOccurrencesOptions{
					TriggerID: chi.URLParam(r, "triggerID"),
				},
			}, nil
		})
		if err != nil {
			sharedapi.BadRequest(w, "VALIDATION", err)
			return
		}
		query.PageSize = normalizePageSize(query.PageSize)
		// The path parameter is authoritative even when the pagination cursor is
		// supplied by the client.
		query.Options.TriggerID = chi.URLParam(r, "triggerID")

		triggersOccurrences, err := backend.ListTriggersOccurrences(r.Context(), *query)
		if err != nil {
			sharedapi.InternalServerError(w, r, err)
			return
		}

		renderCursor(w, *triggersOccurrences)
	}
}
