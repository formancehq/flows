package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sharedapi "github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/auth"
	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestResumeActivity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		stage            string
		activityID       string
		body             string
		backendCalled    bool
		backendError     error
		expectedPausedAt time.Time
		status           int
		errorCode        string
	}{
		{name: "success", stage: "2", backendCalled: true, status: http.StatusNoContent},
		{name: "second precision", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00Z"}`, expectedPausedAt: time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC), backendCalled: true, status: http.StatusNoContent},
		{name: "timezone offset", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T12:00:00.123456789+02:00"}`, backendCalled: true, status: http.StatusNoContent},
		{name: "stage zero", stage: "0", backendCalled: true, status: http.StatusNoContent},
		{name: "instance missing", stage: "2", backendCalled: true, backendError: workflow.ErrInstanceNotFound, status: http.StatusNotFound, errorCode: "NOT_FOUND"},
		{name: "activity missing", stage: "2", backendCalled: true, backendError: workflow.ErrActivityNotFound, status: http.StatusNotFound, errorCode: "NOT_FOUND"},
		{name: "stale run or activity not paused", stage: "2", backendCalled: true, backendError: workflow.ErrActivityResumeConflict, status: http.StatusConflict, errorCode: "CONFLICT"},
		{name: "wrapped instance missing", stage: "2", backendCalled: true, backendError: fmt.Errorf("private backend details: %w", workflow.ErrInstanceNotFound), status: http.StatusNotFound, errorCode: "NOT_FOUND"},
		{name: "wrapped activity missing", stage: "2", backendCalled: true, backendError: fmt.Errorf("private backend details: %w", workflow.ErrActivityNotFound), status: http.StatusNotFound, errorCode: "NOT_FOUND"},
		{name: "wrapped conflict", stage: "2", backendCalled: true, backendError: fmt.Errorf("private backend details: %w", workflow.ErrActivityResumeConflict), status: http.StatusConflict, errorCode: "CONFLICT"},
		{name: "stale pause generation", stage: "2", backendCalled: true, backendError: workflow.ErrActivityResumeConflict, status: http.StatusConflict, errorCode: "CONFLICT"},
		{name: "internal failure", stage: "2", backendCalled: true, backendError: errors.New("private backend details"), status: http.StatusInternalServerError, errorCode: "INTERNAL"},
		{name: "negative stage", stage: "-1", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "invalid stage", stage: "bad", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "fractional stage", stage: "1.5", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "overflowing stage", stage: "99999999999999999999999", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "empty activity", stage: "2", activityID: "", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "blank activity", stage: "2", activityID: "%20", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "missing body", stage: "2", body: " ", status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "missing run id", stage: "2", body: `{"pausedAt":"2026-10-06T10:00:00.123456789Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "empty run id", stage: "2", body: `{"temporalRunID":"","pausedAt":"2026-10-06T10:00:00.123456789Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "blank run id", stage: "2", body: `{"temporalRunID":"  ","pausedAt":"2026-10-06T10:00:00.123456789Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "null run id", stage: "2", body: `{"temporalRunID":null,"pausedAt":"2026-10-06T10:00:00.123456789Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "wrong run id type", stage: "2", body: `{"temporalRunID":42,"pausedAt":"2026-10-06T10:00:00.123456789Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "unknown field", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00.123456789Z","extra":true}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "missing pausedAt", stage: "2", body: `{"temporalRunID":"paused-run"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "empty pausedAt", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":""}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "null pausedAt", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":null}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "wrong pausedAt type", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":42}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "invalid pausedAt", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"bad"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "zero pausedAt", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"0001-01-01T00:00:00Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "missing timezone", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "date only", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "invalid date", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-02-30T10:00:00Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "comma fraction", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00,123Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "single digit hour", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T1:00:00Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "invalid offset hour", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00+24:00"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "invalid offset minute", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00+01:60"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "overprecise pausedAt", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00.1234567891Z"}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "malformed JSON", stage: "2", body: `{"temporalRunID":`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "null body", stage: "2", body: `null`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "array body", stage: "2", body: `[]`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "trailing object", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00.123456789Z"}{}`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "trailing null", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00.123456789Z"}null`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "trailing garbage", stage: "2", body: `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00.123456789Z"}garbage`, status: http.StatusBadRequest, errorCode: "VALIDATION"},
		{name: "trailing whitespace", stage: "2", body: "{\"temporalRunID\":\"paused-run\",\"pausedAt\":\"2026-10-06T10:00:00.123456789Z\"} \n", backendCalled: true, status: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			backend := api.NewMockBackend(gomock.NewController(t))
			if tc.backendCalled {
				stage := 2
				if tc.stage == "0" {
					stage = 0
				}
				backend.EXPECT().ResumeActivity(gomock.Any(), "instance", stage, "activity", "paused-run", gomock.Any()).DoAndReturn(
					func(ctx context.Context, instanceID string, stage int, activityID, temporalRunID string, pausedAt time.Time) error {
						require.Equal(t, "request-context", ctx.Value(resumeActivityContextKey{}))
						expectedPausedAt := tc.expectedPausedAt
						if expectedPausedAt.IsZero() {
							expectedPausedAt = time.Date(2026, 10, 6, 10, 0, 0, 123456789, time.UTC)
						}
						require.True(t, expectedPausedAt.Equal(pausedAt), "expected %s, got %s", expectedPausedAt, pausedAt)
						return tc.backendError
					},
				)
			}
			body := tc.body
			if body == "" {
				body = `{"temporalRunID":"paused-run","pausedAt":"2026-10-06T10:00:00.123456789Z"}`
			}
			activityID := tc.activityID
			if activityID == "" && tc.name != "empty activity" {
				activityID = "activity"
			}
			req := httptest.NewRequest(http.MethodPost, "/instances/instance/stages/"+tc.stage+"/activities/"+activityID+"/resume", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.WithValue(t.Context(), resumeActivityContextKey{}, "request-context"))
			rec := httptest.NewRecorder()
			newRouter(backend, auth.NewNoAuth(), false).ServeHTTP(rec, req)

			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.status == http.StatusNoContent {
				require.Empty(t, rec.Body.String())
				return
			}
			require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
			var response sharedapi.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			require.Equal(t, tc.errorCode, response.ErrorCode)
			require.NotEmpty(t, response.ErrorMessage)
			require.NotContains(t, rec.Body.String(), "private backend details")
		})
	}
}

type resumeActivityContextKey struct{}
