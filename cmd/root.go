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
	targetNamespace string
	clusters        []string
	namedClusters   map[string]string
)

var rootCmd = &cobra.Command{
	Use:     "vcluster-argocd-exporter",
	Version: version + " (" + commit + ") " + date,
	RunE: func(cmd *cobra.Command, args []string) error {
		if (namedClusters == nil || len(namedClusters) == 0) && (clusters == nil || len(clusters) == 0) {
			return errors.New("no clusters specified")
		}

		if len(targetNamespace) == 0 {
			return errors.New("no target namespace specified")
		}

		if clusters != nil && len(clusters) > 0 {
			if namedClusters == nil {
				namedClusters = make(map[string]string)
			}
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
	rootCmd.PersistentFlags().StringVarP(&targetNamespace, "namespace", "n", "argocd", "namespace where ArgoCD is installed")
	rootCmd.PersistentFlags().StringSliceVarP(&clusters, "clusters", "c", nil, "clusters to export")
	rootCmd.PersistentFlags().StringToStringVar(&namedClusters, "named-namedClusters", nil, "named clusters to export")
}
