package maas

import (
	"github.com/openshift/hypershift/cmd/cluster/core"

	"github.com/spf13/cobra"
)

func NewDestroyCommand(opts *core.DestroyOptions) *cobra.Command {

	cmd := &cobra.Command{
		Use:          "maas",
		Short:        "Destroys a HostedCluster and its associated infrastructure on MAAS",
		SilenceUsage: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// For now, just return success as MAAS-specific destroy logic is not yet implemented
		// TODO: Implement MAAS-specific cleanup logic
		return nil
	}

	return cmd
}
