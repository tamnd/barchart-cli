package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) activeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "active",
		Short: "Most actively traded US stocks by volume for the latest trading day",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching top %d most active stocks...", n)
			quotes, err := a.client.Active(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(quotes, len(quotes))
		},
	}
}
