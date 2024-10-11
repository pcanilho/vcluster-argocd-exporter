package vcluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/pcanilho/vcluster-argocd-exporter/internal/k8s"
	"gopkg.in/yaml.v3"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	staticFlags = []string{"_", "connect", "--print", "--service-account", "kube-system/my-user", "--cluster-role", "cluster-admin"}
)

type virtualKubeconfigRaw struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Clusters   []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			Server                   string `yaml:"server"`
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		Name string `yaml:"name"`
		User struct {
			Token string `yaml:"token"`
		} `yaml:"user"`
	} `yaml:"users"`
}

type virtualKubeconfig struct {
	Server                   string
	CertificateAuthorityData string
	Token                    string
}

type argoSecretConfig struct {
	BearerToken     string `json:"bearerToken"`
	TlsClientConfig struct {
		CaData   string `json:"caData"`
		Insecure bool   `json:"insecure"`
	} `json:"tlsClientConfig"`
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

func parseVirtualKubeconfig(virtualKubeconfigYAML []byte) (*virtualKubeconfig, error) {
	var vkc virtualKubeconfigRaw
	if err := yaml.Unmarshal(virtualKubeconfigYAML, &vkc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal virtual kubeconfig: %w", err)
	}

	return &virtualKubeconfig{
		Server:                   vkc.Clusters[0].Cluster.Server,
		CertificateAuthorityData: vkc.Clusters[0].Cluster.CertificateAuthorityData,
		Token:                    vkc.Users[0].User.Token,
	}, nil
}

func ExposeVirtualKubeconfigAsSecret(namespace string, clusters map[string]string) error {
	k8sCtl, err := k8s.NewController()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes controller: %w", err)
	}
	for cluster, targetClusterName := range clusters {
		os.Args = append(staticFlags, cluster)
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		cmd.Execute()
		_ = w.Close()
		vClusterKubeconfig, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		vkc, err := parseVirtualKubeconfig(vClusterKubeconfig)
		if err != nil {
			return fmt.Errorf("failed to parse virtual kubeconfig: %w", err)
		}

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
