package v2

import (
	"encoding/json"
	"net/http"

	api2 "github.com/formancehq/orchestration/internal/api"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/orchestration/internal/workflow"
)

func postEventToWorkflowInstance(backend api2.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event := workflow.Event{}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			api.BadRequest(w, "VALIDATION", err)
			return
		}

		if err := backend.PostEvent(r.Context(), instanceID(r), event); err != nil {
			api.InternalServerError(w, r, err)
			return
		}

		api.NoContent(w)
	}
}
