package main

import (
	"log"

	"github.com/pcanilho/vcluster-argocd-exporter/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("failed to export vcluster cluster(s). error: %v", err)
	}
}
