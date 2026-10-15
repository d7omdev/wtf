package cli

import (
	"github.com/spf13/cobra"

	"wtf/internal/prompt"
)

var explainCmd = &cobra.Command{
	Use:          "explain",
	Short:        "Explain what happened and why (2-3 sentences)",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnalysis(prompt.Explain, prompt.Explain)
	},
}
