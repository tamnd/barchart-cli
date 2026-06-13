package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) gainersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gainers",
		Short: "Top gaining US stocks for the latest trading day",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching top %d gainers...", n)
			quotes, err := a.client.Gainers(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(quotes, len(quotes))
		},
	}
}
