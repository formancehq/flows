package operations

import (
	"encoding/json"
	"testing"
	"time"
)

// The API compares the pause timestamp at nanosecond precision. SDK encoding
// must preserve it, otherwise an observed pause cannot be resumed.
func TestResumeActivityPreservesPauseTimestamp(t *testing.T) {
	pausedAt := time.Date(2026, 10, 6, 10, 0, 0, 123456789, time.UTC)
	for name, body := range map[string]any{
		"v1": ResumeActivityRequestBody{TemporalRunID: "child-run", PausedAt: pausedAt},
		"v2": V2ResumeActivityRequestBody{TemporalRunID: "child-run", PausedAt: pausedAt},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]string
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["temporalRunID"] != "child-run" || fields["pausedAt"] != pausedAt.Format(time.RFC3339Nano) {
				t.Fatalf("resume identity lost during SDK encoding: %s", encoded)
			}
		})
	}
}
