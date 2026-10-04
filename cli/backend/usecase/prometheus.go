package usecase

import (
	"fmt"
	"os"
	"strings"

	"github.com/ericsanto/S.H.A.N.K.S/cli/backend/models"
)

func CreateFileScrapingPrometheus(config models.Cluster) error {
	var bufferDatanodes strings.Builder
	var bufferNodeManager strings.Builder
	var bufferNodeExporter strings.Builder

	for _, datanode := range config.Datanodes {
		fmt.Fprintf(&bufferDatanodes, "      - %s:9406\n", datanode.Name)
		fmt.Fprintf(&bufferNodeManager, "      - %s:9407\n", datanode.Name)
		fmt.Fprintf(&bufferNodeExporter, "      - %s:9100\n", datanode.Name)
	}

	yamlFile := fmt.Sprintf(`scrape_configs:
  - job_name: namenode
    static_configs:
      - targets:
      - %s:9404

  - job_name: resourcemanager
    static_configs:
      - targets:
      - %s:9405

  - job_name: datanodes
    static_configs:
      - targets:
%s
  - job_name: nodemanager
    static_configs:
      - targets:
%s
  - job_name: node_exporter
    static_configs:
      - targets:
%s`,
		config.Namenode.Name,
		config.Namenode.Name,
		bufferDatanodes.String(),
		bufferNodeManager.String(),
		bufferNodeExporter.String(),
	)

	return os.WriteFile("new_cluster/master/prometheus.yml", []byte(yamlFile), 0644)
}
