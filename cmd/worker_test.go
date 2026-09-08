package cmd

import (
	"testing"

	"github.com/formancehq/go-libs/v3/service"
	"github.com/formancehq/go-libs/v3/temporal"
	"github.com/stretchr/testify/require"
)

// go-libs registers the max parallel activities flag as a float64. Reading it
// with the wrong accessor silently yields 0, which makes the temporal SDK fall
// back to its own default of 100000 and turns the rate limit into a no-op.
func TestWorkerOptionsReadsMaxParallelActivities(t *testing.T) {
	t.Setenv("TEMPORAL_MAX_PARALLEL_ACTIVITIES", "10")

	cmd := newWorkerCommand()
	require.NoError(t, cmd.Flags().Parse(nil))
	service.BindEnvToFlagSet(cmd.Flags())

	value, err := cmd.Flags().GetFloat64(temporal.TemporalMaxParallelActivitiesFlag)
	require.NoError(t, err)
	require.Equal(t, float64(10), value)

	_, err = workerOptions(cmd)
	require.NoError(t, err)
}
