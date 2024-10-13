// Package cmd contains the command-line interface for the application.
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pcanilho/vcluster-argocd-exporter/internal/vcluster"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "n/a"
	date    = "n/a"
)

// Execute runs the command.
func Execute() error {
	return rootCmd.Execute()
}

var (
	debug           bool
	autoDiscover    bool
	targetNamespace string
	clusters        []string
	namedClusters   map[string]string
)

var rootCmd = &cobra.Command{
	Use:     "vcluster-argocd-exporter",
	Version: version + " (" + commit + ") " + date,
	RunE: func(_ *cobra.Command, _ []string) error {
		logLevel := slog.LevelInfo
		if debug {
			logLevel = slog.LevelDebug
		}

		slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     logLevel,
			AddSource: debug,
		})).With(
			slog.String("targetNamespace", targetNamespace),
			slog.String("clusters", fmt.Sprintf("%v", clusters)),
			slog.String("namedClusters", fmt.Sprintf("%v", namedClusters)),
			slog.Bool("autoDiscover", autoDiscover))

		slogger.Info("Processing...")
		if autoDiscover {
			slogger.Info("Auto discovering clusters...")
			namedClusters = map[string]string{}

			discoveredClusters, err := vcluster.DiscoverClusters(slogger)
			if err != nil {
				return errors.Wrap(err, "failed to discover clusters")
			}
			if len(discoveredClusters) == 0 {
				return errors.New("no clusters discovered")
			}
			slogger.Info(fmt.Sprintf("discovered [%d] clusters", len(discoveredClusters)), slog.String("clusters", fmt.Sprintf("%v", discoveredClusters)))
			clusters = discoveredClusters
		}

		if len(namedClusters) == 0 && len(clusters) == 0 {
			return errors.New("no clusters specified")
		}

		if len(targetNamespace) == 0 {
			return errors.New("no target namespace specified")
		}

		if len(clusters) > 0 {
			for _, cluster := range clusters {
				namedClusters[cluster] = cluster
			}
		}
		slog.Info("Exporting clusters...", slog.String("clusters", fmt.Sprintf("%v", namedClusters)))
		if err := vcluster.ExposeVirtualKubeconfigAsSecret(slogger, targetNamespace, namedClusters); err != nil {
			return errors.Wrap(err, "failed to write virtual kubeconfig")
		}
		slogger.Info("Clusters exported successfully")
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&targetNamespace, "target-namespace", "t", "argocd", "namespace where ArgoCD is installed")
	rootCmd.PersistentFlags().StringSliceVarP(&clusters, "clusters", "c", []string{}, "clusters to export")
	rootCmd.PersistentFlags().StringToStringVar(&namedClusters, "named-cluster", make(map[string]string), "named clusters to export")
	rootCmd.PersistentFlags().BoolVar(&autoDiscover, "auto-discover", false, "auto discover clusters (overrides all other cluster flags)")
}
