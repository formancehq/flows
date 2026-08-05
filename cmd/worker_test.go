package cmd

import (
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/stretchr/testify/require"
)

func TestWorkerOptionsValidatesMaxParallelActivities(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "positive integer", value: "10"},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "fractional", value: "0.5", wantErr: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cmd := newWorkerCommand()
			require.NoError(t, cmd.Flags().Set(temporal.TemporalMaxParallelActivitiesFlag, testCase.value))
			_, err := workerOptions(cmd)
			if testCase.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
