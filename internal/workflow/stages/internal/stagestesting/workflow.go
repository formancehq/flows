package stagestesting

import (
	"testing"
	"time"

	"github.com/formancehq/orchestration/internal/workflow/stages"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type MockedActivity struct {
	Activity any
	Args     []any
	Returns  []any
}

type DelayedCallback struct {
	Fn       func(environment *testsuite.TestWorkflowEnvironment) func()
	Duration time.Duration
}

type WorkflowTestCase[T stages.Stage] struct {
	Stage            T
	MockedActivities []MockedActivity
	DelayedCallbacks []DelayedCallback
	Name             string
	// ExpectedErrorCode, when set, asserts the workflow terminates with an ApplicationError of
	// this type rather than succeeding.
	ExpectedErrorCode string
	// ExpectedActivityCalls pins how many times an activity ran, keyed by registered name. It is
	// what separates a non-retryable failure from a retryable one: both end the workflow, only
	// the non-retryable one does so on the first attempt.
	ExpectedActivityCalls map[string]int
}

func RunWorkflowTest[T stages.Stage](t *testing.T, testCase WorkflowTestCase[T]) {
	t.Run(testCase.Name, func(t *testing.T) {
		t.Parallel()

		testSuite := &testsuite.WorkflowTestSuite{}

		env := testSuite.NewTestWorkflowEnvironment()
		for _, ma := range testCase.MockedActivities {
			env.OnActivity(ma.Activity, ma.Args...).Return(ma.Returns...)
		}
		for _, callback := range testCase.DelayedCallbacks {
			env.RegisterDelayedCallback(callback.Fn(env), callback.Duration)
		}

		var stage T
		env.ExecuteWorkflow(stage.GetWorkflow(), testCase.Stage)
		require.True(t, env.IsWorkflowCompleted())

		if testCase.ExpectedErrorCode != "" {
			var applicationError *temporal.ApplicationError
			require.ErrorAs(t, env.GetWorkflowError(), &applicationError)
			require.Equal(t, testCase.ExpectedErrorCode, applicationError.Type())
		} else {
			require.NoError(t, env.GetWorkflowError())
		}

		for name, calls := range testCase.ExpectedActivityCalls {
			env.AssertActivityNumberOfCalls(t, name, calls)
		}
	})
}

func RunWorkflows[T stages.Stage](t *testing.T, testCases ...WorkflowTestCase[T]) {
	t.Parallel()

	for _, testCase := range testCases {
		RunWorkflowTest(t, testCase)
	}
}
