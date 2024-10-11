package vcluster

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pcanilho/vcluster-argocd-exporter/internal/k8s"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExposeVirtualKubeconfigAsSecret(namespace string, clusters map[string]string) error {
	k8sCtl, err := k8s.NewController()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes controller: %w", err)
	}

	clusterKubeConfigs, err := vclusterConnect(clusters)
	for cluster, targetClusterName := range clusters {
		vkc := clusterKubeConfigs[cluster]
		resource := coreV1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      fmt.Sprintf("vcluster-%s", cluster),
				Namespace: namespace,
				Labels: map[string]string{
					"argocd.argoproj.io/secret-type": "cluster",
				},
			},
			Type: "opaque",
			StringData: map[string]string{
				"name":   targetClusterName,
				"server": vkc.Server,
				"config": getArgoConfigAsString(vkc),
			},
		}

		_, err = k8sCtl.CreateSecret(context.Background(), namespace, &resource, metaV1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
	}

	return nil
}

func DiscoverClusters() ([]string, error) {
	return vclusterList()
}

func getArgoConfigAsString(kubeconfig *virtualKubeconfig) string {
	asc := argoSecretConfig{
		BearerToken: kubeconfig.Token,
		TlsClientConfig: struct {
			CaData   string `json:"caData"`
			Insecure bool   `json:"insecure"`
		}{
			CaData:   kubeconfig.CertificateAuthorityData,
			Insecure: false,
		},
	}
	argoSecretConfigString, err := json.Marshal(asc)
	if err != nil {
		log.Fatalf("failed to marshal argo secret config: %v", err)
	}
	return string(argoSecretConfigString)
}