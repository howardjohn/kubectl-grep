package cmd

import (
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/howardjohn/kubectl-grep/pkg"
)

var (
	summary          = false
	clean            = false
	decode           = false
	regex            = ""
	invertRegex      = false
	insensitiveRegex = false
	cleanStatus      = false
	diff             = false
	diffMode         = "line"
	outputFolder     = ""
)

var rootCmd = &cobra.Command{
	Use:          "kubectl-grep",
	Short:        "A plugin to grep Kubernetes resources.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dm := pkg.Full
		if summary {
			dm = pkg.Summary
		} else if cleanStatus {
			dm = pkg.CleanStatus
		} else if clean {
			dm = pkg.Clean
		}
		dfm := pkg.DiffLine
		switch diffMode {
		case "line":
			dfm = pkg.DiffLine
		case "inline":
			dfm = pkg.DiffInline
		}
		selector := pkg.Selector{Resources: ParseArgs(args)}
		if regex != "" {
			if insensitiveRegex {
				regex = `(?i)` + regex
			}
			rx, err := regexp.Compile(regex)
			if err != nil {
				return err
			}
			selector.Regex = rx
			selector.InvertRegex = invertRegex
		}
		if outputFolder != "" {
			if err := os.MkdirAll(outputFolder, os.ModePerm); err != nil {
				return err
			}
		}
		opts := pkg.Opts{
			Sel:          selector,
			Mode:         dm,
			Diff:         diff,
			DiffType:     dfm,
			Decode:       decode,
			OutputFolder: outputFolder,
		}
		if err := pkg.GrepResources(opts, cmd.InOrStdin(), cmd.OutOrStdout()); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&summary, "summary", "s", summary,
		"Summarize output")
	rootCmd.PersistentFlags().BoolVarP(&clean, "clean", "n", clean,
		"Cleanup generate fields")
	rootCmd.PersistentFlags().BoolVarP(&decode, "decode", "d", decode,
		"Decode base64 fields in Secrets")

	rootCmd.PersistentFlags().StringVarP(&regex, "regex", "r", regex,
		"Raw regex to match against")
	rootCmd.PersistentFlags().BoolVarP(&invertRegex, "invert-regex", "v", invertRegex,
		"Invert regex match")
	rootCmd.PersistentFlags().BoolVarP(&insensitiveRegex, "insensitive-regex", "i", insensitiveRegex,
		"Invert regex match")

	rootCmd.PersistentFlags().BoolVarP(&cleanStatus, "clean-status", "N", cleanStatus,
		"Cleanup generate fields, including status")

	rootCmd.PersistentFlags().BoolVarP(&diff, "diff", "w", diff,
		"Show diff of changes. Use with 'kubectl -ojson -w | kubectl grep -w'")
	rootCmd.PersistentFlags().StringVar(&diffMode, "diff-mode", diffMode,
		"Format for diffs. Can be [line, inline].")
	rootCmd.PersistentFlags().StringVarP(&outputFolder, "output-folder", "o", outputFolder,
		"Output separate files to given folder")
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
