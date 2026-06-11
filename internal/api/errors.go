package api

import (
	"database/sql"
	"net/http"

	sharedapi "github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/pkg/errors"
	"go.temporal.io/api/serviceerror"
)

// WriteError maps a backend error to the appropriate HTTP response:
//   - 404 for not-found errors: sql.ErrNoRows, the workflow not-found
//     sentinels, and Temporal NotFound (raised when reading the history of an
//     unknown instance/stage);
//   - 400 for invalid workflow configuration;
//   - 500 otherwise.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var temporalNotFound *serviceerror.NotFound
	switch {
	case errors.As(err, &temporalNotFound),
		errors.Is(err, sql.ErrNoRows),
		errors.Is(err, workflow.ErrInstanceNotFound),
		errors.Is(err, workflow.ErrWorkflowNotFound):
		sharedapi.NotFound(w, err)
	case errors.Is(err, workflow.ErrInvalidConfig):
		sharedapi.BadRequest(w, "VALIDATION", err)
	default:
		sharedapi.InternalServerError(w, r, err)
	}
}
