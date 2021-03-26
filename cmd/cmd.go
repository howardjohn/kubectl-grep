package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/howardjohn/kubectl-grep/pkg"
	"github.com/spf13/cobra"
)

var (
	unlist      = false
	summary     = false
	clean       = false
	cleanStatus = false
)

var rootCmd = &cobra.Command{
	Use:   "kubectl-grep",
	Short: "A plugin to grep Kubernetes resources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dm := pkg.Full
		if summary {
			dm = pkg.Summary
		} else if cleanStatus {
			dm = pkg.CleanStatus
		} else if clean {
			dm = pkg.Clean
		}
		resources, err := pkg.GrepResources(ParseArgs(args), cmd.InOrStdin(), dm)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), resources)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&unlist, "unlist", "L", unlist,
		"Split Kubernetes lists")
	rootCmd.PersistentFlags().BoolVarP(&summary, "summary", "s", summary,
		"Summarize output")
	rootCmd.PersistentFlags().BoolVarP(&clean, "clean", "n", clean,
		"Cleanup generate fields")
	rootCmd.PersistentFlags().BoolVarP(&cleanStatus, "clean-status", "N", cleanStatus,
		"Cleanup generate fields, including status")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func ParseArgs(args []string) []pkg.Resource {
	result := []pkg.Resource{}
	for _, arg := range args {
		resource := pkg.Resource{}
		tsplit := strings.Split(arg, "/")
		if len(tsplit) == 2 {
			resource.Kind = tsplit[0]
			arg = tsplit[1]
		} else {
			arg = tsplit[0]
		}

		nsplit := strings.Split(arg, ".")
		if len(nsplit) == 2 {
			resource.Name = nsplit[0]
			resource.Namespace = nsplit[1]
		} else {
			resource.Name = nsplit[0]
		}
		result = append(result, resource)
	}
	return result
}
