package v2

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/orchestration/internal/api"
	"github.com/go-chi/chi/v5"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestCreateWorkflow(t *testing.T) {
	test(t, func(router *chi.Mux, m api.Backend, db *bun.DB) {
		req := httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewBufferString(`{"stages": []}`))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	})
}

func TestCreateWorkflowValidationError(t *testing.T) {
	test(t, func(router *chi.Mux, m api.Backend, db *bun.DB) {
		// An empty stage specification fails config validation; the API must
		// answer 400, not 500.
		req := httptest.NewRequest(http.MethodPost, "/workflows", bytes.NewBufferString(`{"stages": [{}]}`))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
	})
}
