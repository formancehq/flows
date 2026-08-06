package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestListTriggersOccurrencesIsBounded(t *testing.T) {
	ctx := logging.TestingContext()

	test(t, func(router *chi.Mux, _ api.Backend, db *bun.DB) {
		workflowModel := workflow.New(workflow.Config{})
		_, err := db.NewInsert().Model(&workflowModel).Exec(ctx)
		require.NoError(t, err)

		trigger, err := triggers.NewTrigger(triggers.TriggerData{
			Event:      "TEST_EVENT",
			WorkflowID: workflowModel.ID,
		})
		require.NoError(t, err)
		_, err = db.NewInsert().Model(trigger).Exec(ctx)
		require.NoError(t, err)

		for i := 0; i < 20; i++ {
			occurrence := triggers.NewTriggerOccurrence(
				uuid.NewString(),
				trigger.ID,
				publish.EventMessage{Type: "TEST_EVENT", Version: "v1"},
				time.Now().Add(time.Duration(i)*time.Second),
			)
			_, err = db.NewInsert().Model(&occurrence).Exec(ctx)
			require.NoError(t, err)
		}

		type occurrencePage struct {
			Data     []triggers.Occurrence `json:"data"`
			PageSize int                   `json:"pageSize"`
			HasMore  bool                  `json:"hasMore"`
			Next     string                `json:"next"`
		}
		requestPage := func(query string) occurrencePage {
			req := httptest.NewRequest(http.MethodGet, "/triggers/"+trigger.ID+"/occurrences"+query, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Result().StatusCode)
			var page occurrencePage
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&page))
			return page
		}

		for _, testCase := range []struct {
			name     string
			query    string
			expected int
		}{
			{name: "default page size", expected: 15},
			{name: "explicit page size", query: "?pageSize=5", expected: 5},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				page := requestPage(testCase.query)
				require.Len(t, page.Data, testCase.expected)
			})
		}

		firstPage := requestPage("")
		require.True(t, firstPage.HasMore)
		require.NotEmpty(t, firstPage.Next)
		secondPage := requestPage("?cursor=" + url.QueryEscape(firstPage.Next))
		require.Len(t, secondPage.Data, 5)
		require.False(t, secondPage.HasMore)
	})
}
