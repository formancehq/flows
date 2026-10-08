package workflow

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigActivityMaxAttemptsValidation(t *testing.T) {
	cases := []struct {
		name     string
		attempts *int
		valid    bool
	}{
		{"omitted", nil, true},
		{"minimum", new(1), true},
		{"custom", new(3), true},
		{"default", new(15), true},
		{"maximum", new(math.MaxInt32), true},
		{"zero", new(0), false},
		{"negative", new(-1), false},
	}
	if strconv.IntSize > 32 {
		overflow := int64(math.MaxInt32) + 1
		cases = append(cases, struct {
			name     string
			attempts *int
			valid    bool
		}{"overflow", new(int(overflow)), false})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{Stages: []RawStage{}, ActivityMaxAttempts: tc.attempts}
			err := config.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "activityMaxAttempts")
			}
		})
	}
}

func TestNewActivityMaxAttemptsJSONRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        int
	}{
		{"default", `{"name":"default","stages":[]}`, 15},
		{"custom", `{"name":"custom","stages":[],"activityMaxAttempts":3}`, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var config Config
			require.NoError(t, json.Unmarshal([]byte(tc.input), &config))
			require.NoError(t, config.Validate())
			created := New(config)
			require.NotNil(t, created.Config.ActivityMaxAttempts)
			require.Equal(t, tc.want, *created.Config.ActivityMaxAttempts)
			encoded, err := json.Marshal(created)
			require.NoError(t, err)
			var read Workflow
			require.NoError(t, json.Unmarshal(encoded, &read))
			require.NotNil(t, read.Config.ActivityMaxAttempts)
			require.Equal(t, tc.want, *read.Config.ActivityMaxAttempts)
			require.Equal(t, config.Name, read.Config.Name)
			require.Equal(t, config.Stages, read.Config.Stages)
			if tc.name == "default" {
				require.Nil(t, config.ActivityMaxAttempts)
			}
		})
	}
}
