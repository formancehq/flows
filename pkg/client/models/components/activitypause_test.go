package components

import (
	"encoding/json"
	"testing"
)

func TestActivityPauseStageStatusRoundTrip(t *testing.T) {
	input := []byte(`{"stage":0,"instanceID":"instance","startedAt":"2026-10-06T00:00:00Z","pendingActivities":[{"activityID":"activity","temporalRunID":"run","name":"CreateTransaction","attempt":15,"lastFailure":"failed","lastFailureType":"INSUFFICIENT_FUND","reason":"ACTIVITY_ATTEMPT_LIMIT:15","maxAttempts":15}]}`)
	for _, tc := range []struct {
		name  string
		model interface{}
	}{
		{"v1", &StageStatus{}},
		{"v2", &V2StageStatus{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal(input, tc.model); err != nil {
				t.Fatal(err)
			}
			output, err := json.Marshal(tc.model)
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				PendingActivities []struct {
					ActivityID      string `json:"activityID"`
					TemporalRunID   string `json:"temporalRunID"`
					Attempt         int64  `json:"attempt"`
					LastFailureType string `json:"lastFailureType"`
					Reason          string `json:"reason"`
					MaxAttempts     int64  `json:"maxAttempts"`
				} `json:"pendingActivities"`
			}
			if err := json.Unmarshal(output, &decoded); err != nil {
				t.Fatal(err)
			}
			if len(decoded.PendingActivities) != 1 {
				t.Fatalf("expected one paused activity: %s", output)
			}
			pause := decoded.PendingActivities[0]
			if pause.ActivityID != "activity" || pause.TemporalRunID != "run" || pause.Attempt != 15 || pause.MaxAttempts != 15 || pause.Reason != "ACTIVITY_ATTEMPT_LIMIT:15" || pause.LastFailureType != "INSUFFICIENT_FUND" {
				t.Fatalf("pause metadata lost during round trip: %s", output)
			}
		})
	}
}

func TestActivityPauseHistoryRoundTrip(t *testing.T) {
	input := []byte(`{"name":"CreateTransaction","input":{},"startedAt":"2026-10-06T00:00:00Z","terminated":false,"attempt":15,"activityID":"activity","temporalRunID":"run","paused":true,"pauseReason":"ACTIVITY_ATTEMPT_LIMIT:15","lastFailureType":"INSUFFICIENT_FUND"}`)
	for _, tc := range []struct {
		name  string
		model interface{}
	}{
		{"v1", &WorkflowInstanceHistoryStage{}},
		{"v2", &V2WorkflowInstanceHistoryStage{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal(input, tc.model); err != nil {
				t.Fatal(err)
			}
			output, err := json.Marshal(tc.model)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(output, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded["activityID"] != "activity" || decoded["temporalRunID"] != "run" || decoded["paused"] != true || decoded["pauseReason"] != "ACTIVITY_ATTEMPT_LIMIT:15" || decoded["lastFailureType"] != "INSUFFICIENT_FUND" {
				t.Fatalf("history metadata lost during round trip: %s", output)
			}
		})
	}
}

func TestActivityPauseUnavailableRoundTrip(t *testing.T) {
	input := []byte(`{"stage":0,"instanceID":"instance","startedAt":"2026-10-06T00:00:00Z","pauseStateUnavailable":true}`)
	for _, tc := range []struct {
		name  string
		model interface{}
	}{
		{"v1", &StageStatus{}},
		{"v2", &V2StageStatus{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal(input, tc.model); err != nil {
				t.Fatal(err)
			}
			output, err := json.Marshal(tc.model)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]interface{}
			if err := json.Unmarshal(output, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded["pauseStateUnavailable"] != true {
				t.Fatalf("unavailable state lost: %s", output)
			}
			if _, ok := decoded["pendingActivities"]; ok {
				t.Fatalf("unavailable state must omit pending activities: %s", output)
			}
		})
	}
}
