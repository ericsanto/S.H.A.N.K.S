package tui

import (
	"strings"
)

type Model struct {
	NodesHealth []NodeHealth
	selected    int
	HDFSGeral   HDFS
}

type HDFS struct {
	HDFSTotal          string
	HDFSUsed           string
	HDFSUsedPercentage string
}

type NodeHealth struct {
	Name string

	DockerStatus string
	CPU          string
	RAMUsed      string
	RAMTotal     string

	HDFS HDFS

	YARNStatus      string
	YARNMemoryUsed  string
	YARNMemoryTotal string
	Containers      string
}

// func (n *NodeHealth) RAMUsage() float64 {
// 	if n.RAMTotal == 0 {
// 		return 0
// 	}
// 	return n.RAMUsed / n.RAMTotal * 100
// }

func (n *Model) GetHDFSData(data string) *Model {

	// newData := make(map[string]string)

	configuredCapacity := getLineValue(data, "Configured Capacity")
	dfsUsed := getLineValue(data, "DFS Used")
	dfsUsedPercentage := getLineValue(data, "DFS Used%")

	_, hdfsTotalFormatedGB, _ := strings.Cut(configuredCapacity, "(")
	_, dfUsedPercentageGB, _ := strings.Cut(dfsUsedPercentage, ":")
	_, dfsUsedGB, _ := strings.Cut(dfsUsed, "(")

	newHdfsTotal := strings.Replace(hdfsTotalFormatedGB, ")", "", 1)
	newHdfsUsed := strings.Replace(dfsUsedGB, ")", "", 1)

	n.HDFSGeral.HDFSTotal = newHdfsTotal
	n.HDFSGeral.HDFSUsed = newHdfsUsed
	n.HDFSGeral.HDFSUsedPercentage = dfUsedPercentageGB

	return n
}

func (n *NodeHealth) GetHDFSDataNode(data string, namenode string) {

	_, data, _ = strings.Cut(data, namenode)

	configuredCapacity := getLineValue(data, "Configured Capacity")
	dfsUsed := getLineValue(data, "DFS Used")
	dfsUsedPercentage := getLineValue(data, "DFS Used%")

	_, hdfsTotalFormatedGB, _ := strings.Cut(configuredCapacity, "(")
	_, dfUsedPercentageGB, _ := strings.Cut(dfsUsedPercentage, ":")
	_, dfsUsedGB, _ := strings.Cut(dfsUsed, "(")

	newHdfsTotal := strings.Replace(hdfsTotalFormatedGB, ")", "", 1)
	newHdfsUsed := strings.Replace(dfsUsedGB, ")", "", 1)

	n.HDFS.HDFSTotal = newHdfsTotal
	n.HDFS.HDFSUsed = newHdfsUsed
	n.HDFS.HDFSUsedPercentage = dfUsedPercentageGB
}

func (n *NodeHealth) GetYARNData(data string) *NodeHealth {

	_, after, _ := strings.Cut(data, "Total Nodes")
	_, totalNodes, _ := strings.Cut(after, "Active Nodes")
	_, activeNodes, _ := strings.Cut(totalNodes, "Lost Nodes")
	_, lostNodes, _ := strings.Cut(activeNodes, "Unhealthy Nodes")
	// _, unhealthyNodes, _ := strings.Cut(lostNodes, "Decommissioned Nodes")
	// _, decommissionedNodes, _ := strings.Cut(unhealthyNodes, "Rebooted Nodes")
	// _, rebootedNodes, _ := strings.Cut(decommissionedNodes, "Shutdown Nodes")
	// _, shutdownNodes, _ := strings.Cut(rebootedNodes, "Node-Labels")

	n.YARNMemoryUsed = strings.TrimSpace(totalNodes)
	n.YARNMemoryTotal = strings.TrimSpace(activeNodes)
	n.Containers = strings.TrimSpace(lostNodes)

	return nil
}

func (n *NodeHealth) GetDockerData(data string) *NodeHealth {
	n.DockerStatus = strings.ToUpper(getLineValue(data, "status:"))
	n.CPU = getLineValue(data, "CPU:")

	memory := getLineValue(data, "Memory:")
	if used, total, found := strings.Cut(memory, "/"); found {
		n.RAMUsed = strings.TrimSpace(used)
		n.RAMTotal = strings.TrimSpace(total)
	}

	return n
}

func getLineValue(data, prefix string) string {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if value, found := strings.CutPrefix(line, prefix); found {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
