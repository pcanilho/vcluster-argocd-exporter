// Package vcluster wrapper.go provides a wrapper and middleware between this application and the upstream vcluster package.
package vcluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"

	"github.com/pcanilho/vcluster-argocd-exporter/internal/k8s"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExposeVirtualKubeconfigAsSecret exposes the virtual kubeconfig as a secret.
func ExposeVirtualKubeconfigAsSecret(slogger *slog.Logger, namespace string, clusters map[string]string) error {
	k8sCtl, err := k8s.NewController()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes controller: %w", err)
	}
	slogger.Debug("Kubernetes controller created...")
	clusterKubeConfigs, err := vclusterConnect(slogger, clusters)
	if err != nil {
		return fmt.Errorf("failed to connect to virtual clusters: %w", err)
	}
	for cluster, targetClusterName := range clusters {
		slogger.Debug("Processing...", slog.String("cluster", cluster), slog.String("targetClusterName", targetClusterName))
		vkc := clusterKubeConfigs[cluster]
		resource := coreV1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      fmt.Sprintf("vcluster-%s", cluster),
				Namespace: namespace,
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "cluster",
					"managed-by":                     "vcluster-argocd-exporter",
				},
			},
			Type: "opaque",
			StringData: map[string]string{
				"name":   targetClusterName,
				"server": vkc.Server,
				"config": getArgoConfigAsString(slogger, vkc),
			},
		}

		slogger.Debug("Creating secret...", slog.String("name", resource.Name), slog.String("namespace", resource.Namespace))
		_, err = k8sCtl.CreateSecret(context.Background(), namespace, &resource, metaV1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
	}

	return nil
}

// DiscoverClusters discovers the virtual clusters.
func DiscoverClusters(slogger *slog.Logger) ([]string, error) {
	return vclusterList(slogger)
}

func getArgoConfigAsString(slogger *slog.Logger, kubeconfig *virtualKubeconfig) string {
	slogger.Debug("Creating argo secret config...")
	asc := argoSecretConfig{
		BearerToken: kubeconfig.Token,
		TLSClientConfig: struct {
			CaData   string `json:"caData"`
			Insecure bool   `json:"insecure"`
		}{
			CaData:   kubeconfig.CertificateAuthorityData,
			Insecure: false,
		},
	}
	slogger.Debug("Marshalling argo secret config...")
	argoSecretConfigString, err := json.Marshal(asc)
	if err != nil {
		log.Fatalf("failed to marshal argo secret config: %v", err)
	}
	return string(argoSecretConfigString)
}
