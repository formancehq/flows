package workflow

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	common "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	failure "go.temporal.io/api/failure/v1"
	history "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	temporalworkflow "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type stageHistoryIterator struct {
	events []*history.HistoryEvent
	err    error
}

func (i *stageHistoryIterator) HasNext() bool { return len(i.events) > 0 || i.err != nil }

func (i *stageHistoryIterator) Next() (*history.HistoryEvent, error) {
	if i.err != nil {
		err := i.err
		i.err = nil
		return nil, err
	}
	event := i.events[0]
	i.events = i.events[1:]
	return event, nil
}

func stageHistoryScheduled(id int64, activityID, name string) *history.HistoryEvent {
	return &history.HistoryEvent{
		EventId: id, EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
		EventTime: timestamppb.New(time.Unix(id, 0)),
		Attributes: &history.HistoryEvent_ActivityTaskScheduledEventAttributes{
			ActivityTaskScheduledEventAttributes: &history.ActivityTaskScheduledEventAttributes{
				ActivityId: activityID, ActivityType: &common.ActivityType{Name: name},
				Input: &common.Payloads{Payloads: []*common.Payload{{Data: []byte(`{"value":"input"}`)}}},
			},
		},
	}
}

func stageHistoryManager(t *testing.T, pending []*temporalworkflow.PendingActivityInfo, iterator *stageHistoryIterator) *WorkflowManager {
	t.Helper()
	client := &mocks.Client{}
	client.On("DescribeWorkflowExecution", mock.Anything, "instance-0", "").Return(
		&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &temporalworkflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "instance-0", RunId: "described-run"},
			},
			PendingActivities: pending,
		}, nil).Once()
	client.On("GetWorkflowHistory", mock.Anything, "instance-0", "described-run", false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).Return(iterator).Once()
	t.Cleanup(func() { client.AssertExpectations(t) })
	return NewManager(nil, client, "", "", false)
}

func TestReadStageHistoryPendingActivities(t *testing.T) {
	t.Parallel()
	next := time.Unix(100, 0)
	pending := []*temporalworkflow.PendingActivityInfo{
		{
			ActivityId: "retrying", Attempt: 3,
			ScheduledTime:           timestamppb.New(time.Unix(1, 0)),
			NextAttemptScheduleTime: timestamppb.New(next),
		},
		{
			ActivityId: "paused", Attempt: 15, Paused: true,
			ScheduledTime: timestamppb.New(next), NextAttemptScheduleTime: timestamppb.New(next),
			LastFailure: &failure.Failure{Message: "insufficient funds", FailureInfo: &failure.Failure_ApplicationFailureInfo{
				ApplicationFailureInfo: &failure.ApplicationFailureInfo{Type: "INSUFFICIENT_FUND"},
			}},
			PauseInfo: &temporalworkflow.PendingActivityInfo_PauseInfo{
				PausedBy: &temporalworkflow.PendingActivityInfo_PauseInfo_Manual_{
					Manual: &temporalworkflow.PendingActivityInfo_PauseInfo_Manual{Reason: "activity attempt limit reached"},
				},
			},
		},
	}
	manager := stageHistoryManager(t, pending, &stageHistoryIterator{events: []*history.HistoryEvent{
		stageHistoryScheduled(1, "paused", "CreateTransaction"),
		stageHistoryScheduled(2, "retrying", "OtherActivity"),
	}})
	entries, err := manager.ReadStageHistory(t.Context(), "instance", 0)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	paused, retrying := entries[0], entries[1]
	require.Equal(t, "paused", paused.ActivityID)
	require.Equal(t, "described-run", paused.TemporalRunID)
	require.True(t, paused.Paused)
	require.False(t, paused.Terminated)
	require.Nil(t, paused.TerminatedAt)
	require.Equal(t, 15, paused.Attempt)
	require.Equal(t, "activity attempt limit reached", paused.PauseReason)
	require.Equal(t, "insufficient funds", paused.LastFailure)
	require.Equal(t, "INSUFFICIENT_FUND", paused.LastFailureType)
	require.Nil(t, paused.NextExecution)
	require.False(t, retrying.Paused)
	require.Equal(t, 3, retrying.Attempt)
	require.True(t, next.Equal(*retrying.NextExecution))
	encoded, err := json.Marshal(paused)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(encoded, &body))
	require.Equal(t, true, body["paused"])
	require.Equal(t, "paused", body["activityID"])
	require.Equal(t, "described-run", body["temporalRunID"])
	require.Equal(t, "INSUFFICIENT_FUND", body["lastFailureType"])
	require.NotContains(t, body, "nextExecution")
}

func TestReadStageHistoryOptionalPendingMetadata(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		pending *temporalworkflow.PendingActivityInfo
		reason  string
	}{
		{name: "paused without metadata", pending: &temporalworkflow.PendingActivityInfo{ActivityId: "activity", Paused: true}},
		{name: "running without next retry", pending: &temporalworkflow.PendingActivityInfo{
			ActivityId: "activity", ScheduledTime: timestamppb.New(time.Unix(1, 0)),
			LastFailure: &failure.Failure{Message: "timeout", FailureInfo: &failure.Failure_TimeoutFailureInfo{
				TimeoutFailureInfo: &failure.TimeoutFailureInfo{TimeoutType: enums.TIMEOUT_TYPE_START_TO_CLOSE},
			}},
		}},
		{name: "rule pause", reason: "rule limit", pending: &temporalworkflow.PendingActivityInfo{
			ActivityId: "activity", Paused: true,
			PauseInfo: &temporalworkflow.PendingActivityInfo_PauseInfo{PausedBy: &temporalworkflow.PendingActivityInfo_PauseInfo_Rule_{
				Rule: &temporalworkflow.PendingActivityInfo_PauseInfo_Rule{Reason: "rule limit"},
			}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := stageHistoryManager(t, []*temporalworkflow.PendingActivityInfo{tc.pending},
				&stageHistoryIterator{events: []*history.HistoryEvent{stageHistoryScheduled(1, "activity", "Activity")}})
			entries, err := manager.ReadStageHistory(t.Context(), "instance", 0)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			require.Equal(t, tc.pending.Paused, entries[0].Paused)
			require.Equal(t, tc.reason, entries[0].PauseReason)
			require.Empty(t, entries[0].LastFailureType)
			require.Nil(t, entries[0].NextExecution)
		})
	}
}

func TestReadStageHistoryInterleavedCompletion(t *testing.T) {
	t.Parallel()
	manager := stageHistoryManager(t, nil, &stageHistoryIterator{events: []*history.HistoryEvent{
		stageHistoryScheduled(1, "transaction", "CreateTransaction"),
		stageHistoryScheduled(2, "failed", "OtherActivity"),
		{EventId: 3, EventType: enums.EVENT_TYPE_ACTIVITY_TASK_FAILED, EventTime: timestamppb.New(time.Unix(3, 0)),
			Attributes: &history.HistoryEvent_ActivityTaskFailedEventAttributes{ActivityTaskFailedEventAttributes: &history.ActivityTaskFailedEventAttributes{
				ScheduledEventId: 2, Failure: &failure.Failure{Message: "failed"},
			}}},
		{EventId: 4, EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED, EventTime: timestamppb.New(time.Unix(4, 0)),
			Attributes: &history.HistoryEvent_ActivityTaskCompletedEventAttributes{ActivityTaskCompletedEventAttributes: &history.ActivityTaskCompletedEventAttributes{
				ScheduledEventId: 1, Result: &common.Payloads{Payloads: []*common.Payload{{Data: []byte(`{"data":{"id":42}}`)}}},
			}}},
	}})
	entries, err := manager.ReadStageHistory(t.Context(), "instance", 0)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.True(t, entries[0].Terminated)
	require.Empty(t, entries[0].Error)
	require.True(t, time.Unix(4, 0).Equal(*entries[0].TerminatedAt))
	output := entries[0].Output["CreateTransaction"].(map[string]any)
	transaction := output["data"].([]any)[0].(map[string]any)
	require.Equal(t, float64(42), transaction["txid"])
	require.True(t, entries[1].Terminated)
	require.Equal(t, "failed", entries[1].Error)
	require.True(t, time.Unix(3, 0).Equal(*entries[1].TerminatedAt))
}

func TestReadStageHistoryReadErrors(t *testing.T) {
	t.Parallel()
	t.Run("missing execution", func(t *testing.T) {
		client := &mocks.Client{}
		client.On("DescribeWorkflowExecution", mock.Anything, "instance-0", "").Return(nil, serviceerror.NewNotFound("missing")).Once()
		manager := NewManager(nil, client, "", "", false)
		_, err := manager.ReadStageHistory(t.Context(), "instance", 0)
		require.ErrorIs(t, err, ErrInstanceNotFound)
		client.AssertExpectations(t)
	})
	t.Run("history unavailable", func(t *testing.T) {
		expected := errors.New("history unavailable")
		manager := stageHistoryManager(t, nil, &stageHistoryIterator{err: expected})
		_, err := manager.ReadStageHistory(t.Context(), "instance", 0)
		require.ErrorIs(t, err, expected)
	})
}

func TestReadStageHistoryTerminalEventsOverridePendingSnapshot(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		event   *history.HistoryEvent
		message string
	}{
		{name: "canceled", message: "cancelled", event: &history.HistoryEvent{
			EventType:  enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED,
			Attributes: &history.HistoryEvent_ActivityTaskCanceledEventAttributes{ActivityTaskCanceledEventAttributes: &history.ActivityTaskCanceledEventAttributes{ScheduledEventId: 1}},
		}},
		{name: "timed out", message: "timeout", event: &history.HistoryEvent{
			EventType:  enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT,
			Attributes: &history.HistoryEvent_ActivityTaskTimedOutEventAttributes{ActivityTaskTimedOutEventAttributes: &history.ActivityTaskTimedOutEventAttributes{ScheduledEventId: 1}},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.event.EventTime = timestamppb.New(time.Unix(2, 0))
			manager := stageHistoryManager(t, []*temporalworkflow.PendingActivityInfo{{
				ActivityId: "activity", Paused: true,
				PauseInfo: &temporalworkflow.PendingActivityInfo_PauseInfo{PausedBy: &temporalworkflow.PendingActivityInfo_PauseInfo_Manual_{
					Manual: &temporalworkflow.PendingActivityInfo_PauseInfo_Manual{Reason: "limit"},
				}},
			}}, &stageHistoryIterator{events: []*history.HistoryEvent{stageHistoryScheduled(1, "activity", "Activity"), tc.event}})
			entries, err := manager.ReadStageHistory(t.Context(), "instance", 0)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			require.True(t, entries[0].Terminated)
			require.Equal(t, tc.message, entries[0].Error)
			require.False(t, entries[0].Paused)
			require.Empty(t, entries[0].PauseReason)
			require.Nil(t, entries[0].NextExecution)
		})
	}
}
