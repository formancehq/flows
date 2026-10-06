package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/logging"
	api2 "github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/go-chi/chi/v5"
)

var resumeActivityTimestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

func resumeActivity(backend api2.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stage, err := strconv.Atoi(chi.URLParam(r, "number"))
		if err != nil || stage < 0 {
			api.BadRequest(w, api.ErrorCodeValidation, errors.New("stage number must be a nonnegative integer"))
			return
		}
		activityID := chi.URLParam(r, "activityID")
		if strings.TrimSpace(activityID) == "" {
			api.BadRequest(w, api.ErrorCodeValidation, errors.New("activityID must not be empty"))
			return
		}
		var body struct {
			TemporalRunID string `json:"temporalRunID"`
			PausedAt      string `json:"pausedAt"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			api.BadRequest(w, api.ErrorCodeValidation, errors.New("body must be a JSON object containing temporalRunID and pausedAt"))
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			api.BadRequest(w, api.ErrorCodeValidation, errors.New("body must contain exactly one JSON object"))
			return
		}
		if strings.TrimSpace(body.TemporalRunID) == "" {
			api.BadRequest(w, api.ErrorCodeValidation, errors.New("temporalRunID must not be empty"))
			return
		}

		pausedAt, err := time.Parse(time.RFC3339Nano, body.PausedAt)
		if err != nil || pausedAt.IsZero() || !resumeActivityTimestampPattern.MatchString(body.PausedAt) {
			api.BadRequest(w, api.ErrorCodeValidation, errors.New("pausedAt must be a nonzero RFC3339 timestamp"))
			return
		}

		if err := backend.ResumeActivity(r.Context(), instanceID(r), stage, activityID, body.TemporalRunID, pausedAt); err != nil {
			switch {
			case errors.Is(err, workflow.ErrInstanceNotFound):
				api.NotFound(w, workflow.ErrInstanceNotFound)
			case errors.Is(err, workflow.ErrActivityNotFound):
				api.NotFound(w, workflow.ErrActivityNotFound)
			case errors.Is(err, workflow.ErrActivityResumeConflict):
				api.WriteErrorResponse(w, http.StatusConflict, "CONFLICT", workflow.ErrActivityResumeConflict)
			default:
				logging.FromContext(r.Context()).Error(err)
				api.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrorInternal, errors.New("unable to resume activity"))
			}
			return
		}
		api.NoContent(w)
	}
}
