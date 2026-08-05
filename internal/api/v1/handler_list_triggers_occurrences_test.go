package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/messaging/publish"
	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	sharedapi "github.com/formancehq/go-libs/v5/pkg/testing/api"
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

		for _, testCase := range []struct {
			name     string
			query    string
			expected int
		}{
			{name: "default page size", expected: 15},
			{name: "explicit page size", query: "?pageSize=5", expected: 5},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/triggers/"+trigger.ID+"/occurrences"+testCase.query, nil)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				require.Equal(t, http.StatusOK, rec.Result().StatusCode)

				occurrences := make([]triggers.Occurrence, 0)
				sharedapi.ReadResponse(t, rec, &occurrences)
				require.Len(t, occurrences, testCase.expected)
			})
		}
	})
}
