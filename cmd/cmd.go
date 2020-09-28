package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/howardjohn/kubectl-grep/pkg"

	"github.com/spf13/cobra"
)

var (
	unlist = false
)

var rootCmd = &cobra.Command{
	Use: "kubectl-grep",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 && !unlist {
			return fmt.Errorf("requires at least %d arg(s), only received %d", 1, len(args))
		}
		return nil
	},
	Short: "A plugin to grep Kubernetes resources.",
	RunE: func(cmd *cobra.Command, args []string) error {
		resources, err := pkg.GrepResources(ParseArgs(args), cmd.InOrStdin())
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
