package v1

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// SEC-94 regression: a workflow definition soft-deleted via DELETE /workflows/{id}
// must not accept new executions through API v1. Product decision 2026-11-16:
// 404 or 410 are both acceptable; historical access and already-running
// instances are preserved (no cancellation).
func TestRunWorkflowOnDeletedWorkflowIsRejected(t *testing.T) {
	test(t, func(router *chi.Mux, m api.Backend, db *bun.DB) {
		ctx := context.Background()

		w, err := m.Create(ctx, workflow.Config{
			Stages: []workflow.RawStage{},
		})
		require.NoError(t, err)

		// Legitimate flow preserved: running a live workflow still works.
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/instances/", w.ID), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Result().StatusCode)

		// Soft-delete the definition.
		req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/workflows/%s/", w.ID), nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)

		// Attack: the same authorized user retries RunWorkflow with the retained ID.
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/workflows/%s/instances/", w.ID), nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		status := rec.Result().StatusCode
		require.NotEqual(t, http.StatusCreated, status,
			"SEC-94: starting a new instance on a deleted workflow must not return 201, got %d", status)
		require.Contains(t, []int{http.StatusNotFound, http.StatusGone}, status,
			"SEC-94: expected 404 or 410 for run on deleted workflow, got %d", status)

		// No new instance row may be created for the deleted definition.
		count, err := db.NewSelect().Model((*workflow.Instance)(nil)).
			Where("workflow_id = ?", w.ID).
			Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, count,
			"SEC-94: deleted workflow must not gain new instances (only the 1 pre-deletion instance expected)")
	})
}
