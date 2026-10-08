package workflow

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	common "go.temporal.io/api/common/v1"
	failure "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/serviceerror"
	temporalworkflow "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/mocks"
)

// This SQL fixture exercises Bun's instance/stage reads without starting a database
// or Temporal server. All Temporal calls are asserted independently.
type instancePauseConnector struct {
	stages     []Stage
	terminated bool
	missing    bool
}

func (c *instancePauseConnector) Connect(context.Context) (driver.Conn, error) { return c, nil }
func (c *instancePauseConnector) Driver() driver.Driver                        { return c }
func (c *instancePauseConnector) Open(string) (driver.Conn, error)             { return c, nil }
func (c *instancePauseConnector) Close() error                                 { return nil }
func (c *instancePauseConnector) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *instancePauseConnector) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}
func (c *instancePauseConnector) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "pending_activities") {
		return nil, errors.New("pause projection must not become a SQL column")
	}
	if strings.Contains(query, `"workflow_instance_stage_statuses"`) {
		rows := &instancePauseRows{columns: []string{"instance_id", "stage", "temporal_run_id", "started_at", "terminated_at"}}
		for _, stage := range c.stages {
			var terminatedAt driver.Value
			if stage.TerminatedAt != nil {
				terminatedAt = *stage.TerminatedAt
			}
			rows.values = append(rows.values, []driver.Value{"instance", int64(stage.Number), stage.TemporalRunID, time.Unix(1, 0), terminatedAt})
		}
		return rows, nil
	}
	if !strings.Contains(query, `"workflow_instances"`) {
		return nil, errors.New("unexpected query")
	}
	rows := &instancePauseRows{columns: []string{"id", "workflow_id", "terminated"}}
	if !c.missing {
		rows.values = [][]driver.Value{{"instance", "workflow", c.terminated}}
	}
	return rows, nil
}

type instancePauseRows struct {
	columns []string
	values  [][]driver.Value
}

func (r *instancePauseRows) Columns() []string { return r.columns }
func (r *instancePauseRows) Close() error      { return nil }
func (r *instancePauseRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}

func instancePauseManager(t *testing.T, fixture *instancePauseConnector, client *mocks.Client) *WorkflowManager {
	t.Helper()
	db := bun.NewDB(sql.OpenDB(fixture), pgdialect.New())
	t.Cleanup(func() { require.NoError(t, db.Close()); client.AssertExpectations(t) })
	return NewManager(db, client, "", "", false)
}

func instancePauseDescription(reason string) *workflowservice.DescribeWorkflowExecutionResponse {
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &temporalworkflow.WorkflowExecutionInfo{Execution: &common.WorkflowExecution{RunId: "actual-child-run"}},
		PendingActivities: []*temporalworkflow.PendingActivityInfo{
			{ActivityId: "running", ActivityType: &common.ActivityType{Name: "Other"}},
			{ActivityId: "paused", ActivityType: &common.ActivityType{Name: "CreateTransaction"}, Paused: true, Attempt: 15,
				LastFailure: &failure.Failure{Message: "failed", FailureInfo: &failure.Failure_ApplicationFailureInfo{
					ApplicationFailureInfo: &failure.ApplicationFailureInfo{Type: "INSUFFICIENT_FUND"},
				}},
				PauseInfo: &temporalworkflow.PendingActivityInfo_PauseInfo{PausedBy: &temporalworkflow.PendingActivityInfo_PauseInfo_Manual_{
					Manual: &temporalworkflow.PendingActivityInfo_PauseInfo_Manual{Reason: reason},
				}},
			},
		},
	}
}

func TestGetInstancePausedActivities(t *testing.T) {
	t.Parallel()
	client := &mocks.Client{}
	client.On("DescribeWorkflowExecution", mock.Anything, "instance-0", "").Return(instancePauseDescription("ACTIVITY_ATTEMPT_LIMIT:15"), nil).Once()
	manager := instancePauseManager(t, &instancePauseConnector{stages: []Stage{
		{Number: 0, TemporalRunID: "parent-run"}, {Number: 1, TerminatedAt: new(time.Unix(2, 0))},
	}}, client)
	instance, err := manager.GetInstance(t.Context(), "instance")
	require.NoError(t, err)
	require.Len(t, instance.Statuses, 2)
	require.Len(t, instance.Statuses[0].PendingActivities, 1)
	require.False(t, instance.Statuses[0].PauseStateUnavailable)
	paused := instance.Statuses[0].PendingActivities[0]
	require.Equal(t, "paused", paused.ActivityID)
	require.Equal(t, "actual-child-run", paused.TemporalRunID)
	require.Equal(t, "CreateTransaction", paused.Name)
	require.Equal(t, 15, paused.Attempt)
	require.Equal(t, "failed", paused.LastFailure)
	require.Equal(t, "INSUFFICIENT_FUND", paused.LastFailureType)
	require.Equal(t, "ACTIVITY_ATTEMPT_LIMIT:15", paused.Reason)
	require.Equal(t, 15, *paused.MaxAttempts)
	require.Empty(t, instance.Statuses[1].PendingActivities)
	encoded, err := json.Marshal(instance)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"pendingActivities":[`)
	require.Contains(t, string(encoded), `"maxAttempts":15`)
}

func TestGetInstancePauseDescribeErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		describeErr error
		unknown     bool
	}{
		{"legacy missing execution", serviceerror.NewNotFound("expired"), false},
		{"describe unavailable", serviceerror.NewUnavailable("private endpoint and credentials"), true},
		{"other describe error", errors.New("internal server details"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &mocks.Client{}
			client.On("DescribeWorkflowExecution", mock.Anything, "instance-0", "").Return(nil, tc.describeErr).Once()
			manager := instancePauseManager(t, &instancePauseConnector{stages: []Stage{{Number: 0}}}, client)
			instance, err := manager.GetInstance(t.Context(), "instance")
			require.NoError(t, err)
			require.NotNil(t, instance)
			require.Empty(t, instance.Statuses[0].PendingActivities)
			if tc.unknown {
				require.True(t, instance.Statuses[0].PauseStateUnavailable)
				encoded, err := json.Marshal(instance)
				require.NoError(t, err)
				require.Contains(t, string(encoded), `"pauseStateUnavailable":true`)
				require.NotContains(t, string(encoded), tc.describeErr.Error())
				require.NotContains(t, string(encoded), `"pendingActivities"`)
			} else {
				require.False(t, instance.Statuses[0].PauseStateUnavailable)
			}
		})
	}
}

func TestGetInstancePauseLimitReason(t *testing.T) {
	t.Parallel()
	for _, reason := range []string{"manual pause", "ACTIVITY_ATTEMPT_LIMIT:", "ACTIVITY_ATTEMPT_LIMIT:0", "ACTIVITY_ATTEMPT_LIMIT:-1", "ACTIVITY_ATTEMPT_LIMIT:15x"} {
		t.Run(reason, func(t *testing.T) {
			client := &mocks.Client{}
			client.On("DescribeWorkflowExecution", mock.Anything, "instance-0", "").Return(instancePauseDescription(reason), nil).Once()
			manager := instancePauseManager(t, &instancePauseConnector{stages: []Stage{{Number: 0}}}, client)
			instance, err := manager.GetInstance(t.Context(), "instance")
			require.NoError(t, err)
			paused := instance.Statuses[0].PendingActivities[0]
			require.Equal(t, reason, paused.Reason)
			require.Nil(t, paused.MaxAttempts)
		})
	}
}

func TestGetInstancePauseUnavailableStageDoesNotBlockOthers(t *testing.T) {
	t.Parallel()
	client := &mocks.Client{}
	client.On("DescribeWorkflowExecution", mock.Anything, "instance-0", "").Return(nil, serviceerror.NewUnavailable("private endpoint")).Once()
	client.On("DescribeWorkflowExecution", mock.Anything, "instance-1", "").Return(instancePauseDescription("ACTIVITY_ATTEMPT_LIMIT:15"), nil).Once()
	manager := instancePauseManager(t, &instancePauseConnector{stages: []Stage{{Number: 0}, {Number: 1}}}, client)
	instance, err := manager.GetInstance(t.Context(), "instance")
	require.NoError(t, err)
	require.True(t, instance.Statuses[0].PauseStateUnavailable)
	require.Empty(t, instance.Statuses[0].PendingActivities)
	require.False(t, instance.Statuses[1].PauseStateUnavailable)
	require.Len(t, instance.Statuses[1].PendingActivities, 1)
}

func TestGetInstancePauseLegacyAndTerminalInstances(t *testing.T) {
	t.Parallel()
	for _, fixture := range []*instancePauseConnector{
		{}, {terminated: true, stages: []Stage{{Number: 0}}}, {missing: true},
	} {
		client := &mocks.Client{}
		manager := instancePauseManager(t, fixture, client)
		instance, err := manager.GetInstance(t.Context(), "instance")
		if fixture.missing {
			require.ErrorIs(t, err, ErrInstanceNotFound)
			require.Nil(t, instance)
		} else {
			require.NoError(t, err)
		}
		client.AssertNotCalled(t, "DescribeWorkflowExecution", mock.Anything, mock.Anything, mock.Anything)
	}
}
