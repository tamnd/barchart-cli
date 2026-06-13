package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) losersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "losers",
		Short: "Top losing US stocks for the latest trading day",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching top %d losers...", n)
			quotes, err := a.client.Losers(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(quotes, len(quotes))
		},
	}
}
