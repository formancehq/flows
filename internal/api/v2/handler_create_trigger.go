package v2

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/orchestration/internal/api"

	sharedapi "github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/pkg/errors"
)

func createTrigger(backend api.Backend) func(writer http.ResponseWriter, request *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		data := triggers.TriggerData{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			sharedapi.InternalServerError(w, r, err)
			return
		}

		trigger, err := backend.CreateTrigger(r.Context(), data)
		if err != nil {
			switch {
			case errors.Is(err, triggers.ErrMissingWorkflowID),
				errors.Is(err, triggers.ErrMissingEvent),
				triggers.IsExprCompilationError(err):
				sharedapi.BadRequest(w, "VALIDATION", err)
			case errors.Is(err, triggers.ErrWorkflowNotExists):
				sharedapi.NotFound(w, err)
			default:
				sharedapi.InternalServerError(w, r, err)
			}
			return
		}

		sharedapi.Created(w, trigger)
	}
}
