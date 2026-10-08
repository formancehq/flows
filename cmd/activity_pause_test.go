package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStagePauseFlags(t *testing.T) {
	for _, newCommand := range []func() *cobra.Command{newWorkerCommand, newServeCommand} {
		command := newCommand()
		enabled, err := command.Flags().GetBool(pauseStageActivitiesFlag)
		require.NoError(t, err)
		require.False(t, enabled)
		limit, err := command.Flags().GetInt(stageActivityAttemptsFlag)
		require.NoError(t, err)
		require.Equal(t, 15, limit)
		for _, value := range []string{"0", "-1", "2147483648"} {
			require.NoError(t, command.Flags().Set(stageActivityAttemptsFlag, value))
			_, err := commonOptions(command)
			require.ErrorContains(t, err, stageActivityAttemptsFlag)
		}
	}
}
