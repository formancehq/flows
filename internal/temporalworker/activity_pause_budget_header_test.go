package temporalworker

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	common "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

func TestWorkflowActivityBudgetRejectsInvalidInheritedHeaders(t *testing.T) {
	for _, value := range []any{"invalid", 0, -1, int64(2147483648)} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{&StagePauseInterceptor{Attempts: 15}}})
			payload, err := converter.GetDefaultDataConverter().ToPayload(value)
			require.NoError(t, err)
			env.SetHeader(&common.Header{Fields: map[string]*common.Payload{workflowAttemptsHeader: payload}})
			ran := false
			env.ExecuteWorkflow(func(workflow.Context) error { ran = true; return nil })
			require.True(t, env.IsWorkflowCompleted())
			require.False(t, ran, "invalid inherited budgets must fail before business work")
			var applicationError *temporal.ApplicationError
			require.True(t, errors.As(env.GetWorkflowError(), &applicationError))
			require.Equal(t, "INVALID_PAUSE_LIMIT", applicationError.Type())
			require.True(t, applicationError.NonRetryable())
		})
	}
}
