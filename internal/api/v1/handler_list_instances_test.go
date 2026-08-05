package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"

	"github.com/go-chi/chi/v5"

	sharedapi "github.com/formancehq/go-libs/v5/pkg/testing/api"

	"github.com/google/uuid"

	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestListInstances(t *testing.T) {
	ctx := logging.TestingContext()

	test(t, func(router *chi.Mux, m api.Backend, db *bun.DB) {
		// Create a workflow with 10 instances
		w := workflow.New(workflow.Config{})
		_, err := db.NewInsert().Model(&w).Exec(ctx)
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			instance := workflow.NewInstance(uuid.NewString(), w.ID)
			if i > 5 {
				instance.SetTerminated(time.Now())
			}
			_, err := db.NewInsert().Model(&instance).Exec(ctx)
			require.NoError(t, err)
		}

		req := httptest.NewRequest(http.MethodGet, "/instances", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Result().StatusCode)

		instances := make([]workflow.Instance, 0)
		sharedapi.ReadResponse(t, rec, &instances)
		require.Len(t, instances, 10)

		// Retrieve only running instances
		req = httptest.NewRequest(http.MethodGet, "/instances?running=true", nil)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Result().StatusCode)
		sharedapi.ReadResponse(t, rec, &instances)
		require.Len(t, instances, 6)

		// Delete the workflow
		req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/workflows/%s/", w.ID), nil)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)

		// Try to retrieve instances for all workflows
		req = httptest.NewRequest(http.MethodGet, "/instances", nil)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Result().StatusCode)
		instances = make([]workflow.Instance, 0)
		sharedapi.ReadResponse(t, rec, &instances)
		require.Len(t, instances, 0)

		// Try to retrieve instances for the deleted workflow
		req = httptest.NewRequest(http.MethodGet, "/instances?workflowID="+w.ID, nil)
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Result().StatusCode)
		instances = make([]workflow.Instance, 0)
		sharedapi.ReadResponse(t, rec, &instances)
		require.Len(t, instances, 0)
	})
}

func TestListInstancesIsBounded(t *testing.T) {
	ctx := logging.TestingContext()

	test(t, func(router *chi.Mux, m api.Backend, db *bun.DB) {
		w := workflow.New(workflow.Config{})
		_, err := db.NewInsert().Model(&w).Exec(ctx)
		require.NoError(t, err)

		for i := 0; i < 20; i++ {
			instance := workflow.NewInstance(uuid.NewString(), w.ID)
			_, err := db.NewInsert().Model(&instance).Exec(ctx)
			require.NoError(t, err)
		}

		// Without a page size the default (15) bounds the result instead of
		// loading the whole table.
		req := httptest.NewRequest(http.MethodGet, "/instances", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Result().StatusCode)
		var firstPage struct {
			Data     []workflow.Instance `json:"data"`
			PageSize int                 `json:"pageSize"`
			HasMore  bool                `json:"hasMore"`
			Next     string              `json:"next"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&firstPage))
		require.Len(t, firstPage.Data, 15)
		require.Equal(t, 15, firstPage.PageSize)
		require.True(t, firstPage.HasMore)
		require.NotEmpty(t, firstPage.Next)

		// The continuation cursor makes the remaining records retrievable.
		req = httptest.NewRequest(http.MethodGet, "/instances?cursor="+url.QueryEscape(firstPage.Next), nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Result().StatusCode)
		var secondPage struct {
			Data    []workflow.Instance `json:"data"`
			HasMore bool                `json:"hasMore"`
		}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&secondPage))
		require.Len(t, secondPage.Data, 5)
		require.False(t, secondPage.HasMore)

		// An explicit page size is honoured.
		req = httptest.NewRequest(http.MethodGet, "/instances?pageSize=5", nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Result().StatusCode)
		instances := make([]workflow.Instance, 0)
		sharedapi.ReadResponse(t, rec, &instances)
		require.Len(t, instances, 5)
	})
}
