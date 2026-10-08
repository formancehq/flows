package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/auth"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/orchestration/internal/api"
	"github.com/formancehq/orchestration/internal/temporalworker"
	"github.com/formancehq/orchestration/internal/triggers"
	"github.com/formancehq/orchestration/internal/workflow"
	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

// Exercise the API and real parent workflows, replacing only the stage's
// business activities with synthetic calls so no Ledger or payment is written.
func TestWorkflowActivityBudgetE2E(t *testing.T) {
	for _, tc := range []struct {
		name         string
		attempts     *int
		budget       int32
		accountCalls int32
	}{
		{name: "omitted-default-15", budget: 15, accountCalls: 1},
		{name: "explicit-1", attempts: new(1), budget: 1, accountCalls: 1},
		{name: "explicit-3-independent-budgets", attempts: new(3), budget: 3, accountCalls: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			test(t, func(_ *chi.Mux, _ api.Backend, db *bun.DB) {
				var accountCalls, transactionCalls atomic.Int32
				var ready atomic.Bool
				var keysMu sync.Mutex
				var transactionKeys []string
				businessActivities := temporalworker.NewDefinitionSet().
					Append(temporalworker.Definition{Name: "GetAccount", Func: func(context.Context) error {
						if accountCalls.Add(1) < tc.accountCalls {
							return temporal.NewApplicationError("synthetic account unavailable", "UNAVAILABLE")
						}
						return nil
					}}).
					Append(temporalworker.Definition{Name: "CreateTransaction", Func: func(ctx context.Context) error {
						info := activity.GetInfo(ctx)
						keysMu.Lock()
						transactionKeys = append(transactionKeys, info.WorkflowExecution.RunID+"-"+info.ActivityID)
						keysMu.Unlock()
						transactionCalls.Add(1)
						if !ready.Load() {
							return temporal.NewApplicationError("synthetic insufficient funds", "INSUFFICIENT_FUND")
						}
						return nil
					}})
				stageWorkflow := func(ctx temporalworkflow.Context, _ stages.NoOp) error {
					ctx = temporalworkflow.WithActivityOptions(ctx, temporalworkflow.ActivityOptions{
						StartToCloseTimeout: time.Second,
						RetryPolicy: &temporal.RetryPolicy{
							InitialInterval:        20 * time.Millisecond,
							BackoffCoefficient:     1,
							MaximumInterval:        20 * time.Millisecond,
							MaximumAttempts:        15,
							NonRetryableErrorTypes: []string{"INSUFFICIENT_FUND"},
						},
					})
					if err := temporalworkflow.ExecuteActivity(ctx, "GetAccount").Get(ctx, nil); err != nil {
						return err
					}
					return temporalworkflow.ExecuteActivity(ctx, "CreateTransaction").Get(ctx, nil)
				}
				queue := uuid.NewString()
				w := temporalworker.New(logging.Testing(), devServer.Client(), queue,
					[]temporalworker.DefinitionSet{
						workflow.NewWorkflows("test", false).DefinitionSet(),
						temporalworker.NewDefinitionSet().Append(temporalworker.Definition{Name: "RunNoOp", Func: stageWorkflow}),
					},
					[]temporalworker.DefinitionSet{
						workflow.NewActivities(publish.NoOpPublisher, db).DefinitionSet(),
						businessActivities,
					},
					worker.Options{Interceptors: []interceptor.WorkerInterceptor{
						&temporalworker.StagePauseInterceptor{Attempts: 15},
					}},
				)
				require.NoError(t, w.Start())
				t.Cleanup(w.Stop)
				manager := workflow.NewManager(db, devServer.Client(), "test", queue, false)
				backend := api.NewDefaultBackend(triggers.NewManager(db, triggers.NewExpressionEvaluator(http.DefaultClient)), manager)
				router := newRouter(backend, auth.NewNoAuth(), testing.Verbose())
				request := func(method, path string, body any) *httptest.ResponseRecorder {
					encoded, err := json.Marshal(body)
					require.NoError(t, err)
					r := httptest.NewRequest(method, path, bytes.NewReader(encoded)).WithContext(t.Context())
					r.Header.Set("Content-Type", "application/json")
					response := httptest.NewRecorder()
					router.ServeHTTP(response, r)
					return response
				}

				createdResponse := request(http.MethodPost, "/workflows", workflow.Config{
					Name: tc.name, Stages: []workflow.RawStage{{"noop": {}}}, ActivityMaxAttempts: tc.attempts,
				})
				require.Equal(t, http.StatusCreated, createdResponse.Code, createdResponse.Body.String())
				var created struct {
					Data workflow.Workflow `json:"data"`
				}
				require.NoError(t, json.Unmarshal(createdResponse.Body.Bytes(), &created))
				require.NotEmpty(t, created.Data.ID)
				require.NotNil(t, created.Data.Config.ActivityMaxAttempts)
				require.Equal(t, int(tc.budget), *created.Data.Config.ActivityMaxAttempts)
				runResponse := request(http.MethodPost, "/workflows/"+created.Data.ID+"/instances", map[string]string{})
				require.Equal(t, http.StatusCreated, runResponse.Code, runResponse.Body.String())
				var run struct {
					Data workflow.Instance `json:"data"`
				}
				require.NoError(t, json.Unmarshal(runResponse.Body.Bytes(), &run))
				require.NotEmpty(t, run.Data.ID)
				readInstance := func() workflow.Instance {
					response := request(http.MethodGet, "/instances/"+run.Data.ID, nil)
					require.Equal(t, http.StatusOK, response.Code, response.Body.String())
					var read struct {
						Data workflow.Instance `json:"data"`
					}
					require.NoError(t, json.Unmarshal(response.Body.Bytes(), &read))
					return read.Data
				}
				var paused workflow.ActivityPause
				require.Eventually(t, func() bool {
					instance := readInstance()
					if instance.Terminated || len(instance.Statuses) != 1 || len(instance.Statuses[0].PendingActivities) != 1 {
						return false
					}
					paused = instance.Statuses[0].PendingActivities[0]
					return paused.Name == "CreateTransaction" && paused.LastFailureType == "INSUFFICIENT_FUND"
				}, 20*time.Second, 20*time.Millisecond)
				require.Equal(t, "ACTIVITY_ATTEMPT_LIMIT:"+fmt.Sprint(tc.budget), paused.Reason)
				require.NotNil(t, paused.MaxAttempts)
				require.Equal(t, int(tc.budget), *paused.MaxAttempts)
				require.NotNil(t, paused.PausedAt)
				require.False(t, paused.PausedAt.IsZero())
				require.NotEmpty(t, paused.TemporalRunID)
				require.NotEmpty(t, paused.ActivityID)
				require.Equal(t, tc.accountCalls, accountCalls.Load())
				require.Equal(t, tc.budget, transactionCalls.Load())
				key := paused.TemporalRunID + "-" + paused.ActivityID
				assertKeys := func(want int) {
					keysMu.Lock()
					defer keysMu.Unlock()
					require.Len(t, transactionKeys, want)
					for _, observed := range transactionKeys {
						require.Equal(t, key, observed)
					}
				}
				assertKeys(int(tc.budget))
				require.Never(t, func() bool {
					return transactionCalls.Load() != tc.budget || accountCalls.Load() != tc.accountCalls
				}, 250*time.Millisecond, 20*time.Millisecond)

				ready.Store(true)
				resumeResponse := request(http.MethodPost, fmt.Sprintf("/instances/%s/stages/0/activities/%s/resume", run.Data.ID, paused.ActivityID), map[string]string{
					"temporalRunID": paused.TemporalRunID,
					"pausedAt":      paused.PausedAt.Format(time.RFC3339Nano),
				})
				require.Equal(t, http.StatusNoContent, resumeResponse.Code, resumeResponse.Body.String())
				var completed workflow.Instance
				require.Eventually(t, func() bool {
					completed = readInstance()
					return completed.Terminated
				}, 20*time.Second, 20*time.Millisecond)
				require.Empty(t, completed.Error)
				require.NotNil(t, completed.TerminatedAt)
				require.Len(t, completed.Statuses, 1)
				require.NotNil(t, completed.Statuses[0].TerminatedAt)
				require.Nil(t, completed.Statuses[0].Error)
				require.Empty(t, completed.Statuses[0].PendingActivities)
				require.Equal(t, tc.accountCalls, accountCalls.Load(), "successful GetAccount must not replay")
				require.Equal(t, tc.budget+1, transactionCalls.Load())
				assertKeys(int(tc.budget + 1))
			})
		})
	}
}
