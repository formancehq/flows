package v1

import (
	"net/http"

	api2 "github.com/formancehq/orchestration/internal/api"

	"github.com/formancehq/go-libs/v2/api"
)

func abortWorkflowInstance(backend api2.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := backend.AbortRun(r.Context(), instanceID(r)); err != nil {
			api.InternalServerError(w, r, err)
			return
		}
		api.NoContent(w)
	}
}
