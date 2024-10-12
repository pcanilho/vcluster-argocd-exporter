// Package vcluster client.go provides an interface to interact with the upstream vcluster package.
package vcluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	logger "github.com/loft-sh/log"
	"github.com/loft-sh/vcluster/cmd/vclusterctl/cmd"
	"github.com/loft-sh/vcluster/pkg/cli/flags"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

type virtualCluster struct {
	Name string `json:"name"`
}

type argoSecretConfig struct {
	BearerToken     string `json:"bearerToken"`
	TLSClientConfig struct {
		CaData   string `json:"caData"`
		Insecure bool   `json:"insecure"`
	} `json:"tlsClientConfig"`
}

func vclusterConnect(slogger *slog.Logger, clusters map[string]string) (map[string]*virtualKubeconfig, error) {
	vkc := make(map[string]*virtualKubeconfig)
	for cluster := range clusters {
		rootCmd := cmd.NewRootCmd(nil)
		rootCmd.SetContext(context.Background())
		persistentFlags := rootCmd.PersistentFlags()
		globalFlags := flags.SetGlobalFlags(persistentFlags, nil)
		connectCmd := cmd.NewConnectCmd(globalFlags)
		_ = connectCmd.Flags().Set("print", "true")
		_ = connectCmd.Flags().Set("service-account", "kube-system/my-user")
		_ = connectCmd.Flags().Set("cluster-role", "cluster-admin")

		vClusterKubeconfig, err := captureStdout(connectCmd.RunE, rootCmd, []string{cluster})
		if err != nil {
			return nil, fmt.Errorf("failed to execute connect command: %w", err)
		}

		vkcParsed, err := parseVirtualKubeconfig(slogger, vClusterKubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse virtual kubeconfig: %w", err)
		}
		vkc[cluster] = vkcParsed
	}
	return vkc, nil
}

func vclusterList(slogger *slog.Logger) ([]string, error) {
	var buf bytes.Buffer
	logger.Default = logger.NewStdoutLogger(os.Stdin, &buf, os.Stderr, logrus.InfoLevel)
	rootCmd := cmd.NewRootCmd(nil)
	rootCmd.SetContext(context.Background())
	persistentFlags := rootCmd.PersistentFlags()
	globalFlags := flags.SetGlobalFlags(persistentFlags, nil)
	listCmd := cmd.NewListCmd(globalFlags)

	_ = listCmd.Flags().Set("output", "json")
	slogger.Debug("Executing list command...")
	if err := listCmd.RunE(rootCmd, []string{}); err != nil {
		return nil, fmt.Errorf("failed to execute list command: %w", err)
	}

	slogger.Debug("Parsing virtual cluster list...")
	vclNames, err := parseVirtualClusterList(slogger, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse virtual cluster list: %w", err)
	}

	return vclNames, nil
}

func parseVirtualKubeconfig(slogger *slog.Logger, virtualKubeconfigYAML []byte) (*virtualKubeconfig, error) {
	var vkc virtualKubeconfigRaw
	slogger.Debug("Unmarshalling virtual kubeconfig...")
	if err := yaml.Unmarshal(virtualKubeconfigYAML, &vkc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal virtual kubeconfig: %w", err)
	}

	slogger.Debug("Parsing virtual kubeconfig...")
	return &virtualKubeconfig{
		Server:                   vkc.Clusters[0].Cluster.Server,
		CertificateAuthorityData: vkc.Clusters[0].Cluster.CertificateAuthorityData,
		Token:                    vkc.Users[0].User.Token,
	}, nil
}

func parseVirtualClusterList(slogger *slog.Logger, virtualClusterListJSON []byte) ([]string, error) {
	var vcl []virtualCluster
	slogger.Debug("Unmarshalling virtual cluster list...")
	if err := json.Unmarshal(virtualClusterListJSON, &vcl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal virtual cluster list: %w", err)
	}

	vclNames := make([]string, 0)
	for _, vc := range vcl {
		vclNames = append(vclNames, vc.Name)
	}

	slogger.Debug("Parsing virtual cluster list...", slog.String("virtualClusters", fmt.Sprintf("%v", vclNames)))
	return vclNames, nil
}

func captureStdout(f func(*cobra.Command, []string) error, cmd *cobra.Command, args []string) ([]byte, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	os.Stdout = w
	outErr := f(cmd, args)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), outErr
}
