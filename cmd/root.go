package cmd

import (
	"github.com/pcanilho/vcluster-argocd-exporter/internal/vcluster"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "n/a"
	date    = "n/a"
)

func Execute() error {
	return rootCmd.Execute()
}

var (
	autoDiscover    bool
	targetNamespace string
	clusters        []string
	namedClusters   map[string]string
)

var rootCmd = &cobra.Command{
	Use:     "vcluster-argocd-exporter",
	Version: version + " (" + commit + ") " + date,
	RunE: func(cmd *cobra.Command, args []string) error {
		if autoDiscover {
			namedClusters = map[string]string{}

			discoveredClusters, err := vcluster.DiscoverClusters()
			if err != nil {
				return errors.Wrap(err, "failed to discover clusters")
			}
			if len(discoveredClusters) == 0 {
				return errors.New("no clusters discovered")
			}
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
		if err := vcluster.ExposeVirtualKubeconfigAsSecret(targetNamespace, namedClusters); err != nil {
			return errors.Wrap(err, "failed to write virtual kubeconfig")
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&targetNamespace, "target-namespace", "t", "argocd", "namespace where ArgoCD is installed")
	rootCmd.PersistentFlags().StringSliceVarP(&clusters, "clusters", "c", []string{}, "clusters to export")
	rootCmd.PersistentFlags().StringToStringVar(&namedClusters, "named-cluster", make(map[string]string), "named clusters to export")
	rootCmd.PersistentFlags().BoolVar(&autoDiscover, "auto-discover", false, "auto discover clusters (overrides all other cluster flags)")
}
