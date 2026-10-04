package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B9B9B"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5A56E0"))

	healthyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#02BF87"))

	unhealthyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5F87"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777"))

	cpuCollumn = lipgloss.NewStyle().
			Width(20)

	nodeCollumn = lipgloss.NewStyle().
			Width(20)

	statusCollumn = lipgloss.NewStyle().
			Width(20)

	ramCollumn = lipgloss.NewStyle().
			Width(40)

	hdfsTotalCollumn = lipgloss.NewStyle().
				Width(20)

	cursorCollumn = lipgloss.NewStyle().
			Width(2)

	hdfsUsedCollumn = lipgloss.NewStyle().
			Width(20)

	hdfsUsePercentageCollumn = lipgloss.NewStyle().
					Width(20)
)

func renderStatus(status string) string {
	switch status {
	case "RUNNING", "LIVE", "healthy":
		return healthyStyle.Render("● " + status)

	case "DOWN", "DEAD", "LOST", "unhealthy":
		return unhealthyStyle.Render("● " + status)

	default:
		return valueStyle.Render(status)
	}
}
func (m Model) View() tea.View {
	var s strings.Builder

	header := titleStyle.Render("SHANKS HEALTH") + "\n" +
		subtitleStyle.Render("Cluster monitoring dashboard")

	s.WriteString(panelStyle.Render(header))
	s.WriteString("\n")

	healthy := 0
	unhealthy := 0

	for _, node := range m.NodesHealth {
		if node.DockerStatus == "RUNNING" {
			healthy++
		} else if node.DockerStatus == "EXITED" {
			unhealthy++
		}
	}

	// OVERVIEW
	overview := fmt.Sprintf(
		"%s  %d     %s  %s     %s  %s",
		labelStyle.Render("Nodes"),
		len(m.NodesHealth),
		healthyStyle.Render("Healthy"),
		valueStyle.Render(fmt.Sprintf("%d", healthy)),
		unhealthyStyle.Render("Unhealthy"),
		valueStyle.Render(fmt.Sprintf("%d", unhealthy)),
	)

	s.WriteString(panelStyle.Render(overview))
	s.WriteString("\n")
	s.WriteString("\n")
	s.WriteString("\n")

	tableConfigMainCluster := fmt.Sprintf(

		"%-14s %-16s %-16s",
		"HDFS Total",
		"HDFS Used",
		"HDFS Used%",
	)

	s.WriteString(headerStyle.Render(tableConfigMainCluster))
	s.WriteString("\n")

	s.WriteString(
		subtitleStyle.Render(
			"───────────────────────────────────────────",
		),
	)
	s.WriteString("\n")

	valuesHdfs := fmt.Sprintf(

		"%-14s %-16s %-16s",
		m.HDFSGeral.HDFSTotal,
		m.HDFSGeral.HDFSUsed,
		m.HDFSGeral.HDFSUsedPercentage,
	)

	s.WriteString(valuesHdfs)
	s.WriteString("\n")
	s.WriteString("\n")
	s.WriteString("\n")
	s.WriteString("\n")

	// TABLE HEADER
	tableHeader := fmt.Sprintf(
		"%-2s%-20s %-20s %-20s %-40s %-20s %-20s %-20s",
		"",
		"NODE",
		"STATUS",
		"CPU",
		"RAM",
		"HDFS TOTAL",
		"HDFS USE",
		"HDFS USE%",
	)

	s.WriteString(headerStyle.Render(tableHeader))
	s.WriteString("\n")

	s.WriteString(
		subtitleStyle.Render(
			"──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────",
		),
	)
	s.WriteString("\n")

	// NODES
	for i, node := range m.NodesHealth {
		cursor := " "

		if i == m.selected {
			cursor = ">"
		}

		cur := cursorCollumn.Render(cursor)
		name := nodeCollumn.Render(node.Name)
		status := statusCollumn.Render(renderStatus(node.DockerStatus))
		cpu := cpuCollumn.Render(node.CPU)
		hdfsTotal := hdfsTotalCollumn.Render(node.HDFS.HDFSTotal)
		ram := ramCollumn.Render(fmt.Sprintf("%s/%s", node.RAMUsed, node.RAMTotal))
		hdfsUsed := hdfsUsedCollumn.Render(node.HDFS.HDFSUsed)
		hdfsUsedPercentage := hdfsUsePercentageCollumn.Render(node.HDFS.HDFSUsedPercentage)

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			cur,
			name,
			status,
			cpu,
			ram,
			hdfsTotal,
			hdfsUsed,
			hdfsUsedPercentage,
		)

		if i == m.selected {
			line = selectedStyle.
				Padding(0, 1).
				Render(line)
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString("\n")

	// SELECTED NODE DETAILS
	if len(m.NodesHealth) > 0 {
		node := m.NodesHealth[m.selected]

		docker := fmt.Sprintf(
			"%s\n%s %s\n%s %s\n%s %s",
			titleStyle.Render("Docker"),
			labelStyle.Render("Status"),
			renderStatus(node.DockerStatus),
			labelStyle.Render("CPU"),
			valueStyle.Render(node.CPU),
			labelStyle.Render("Memory"),
			valueStyle.Render(
				fmt.Sprintf("%s / %s GB", node.RAMUsed, node.RAMTotal),
			),
		)

		hdfs := fmt.Sprintf(
			"%s\n%s %s\n%s %s\n %s",
			titleStyle.Render("HDFS"),
			labelStyle.Render("Status"),
			renderStatus(node.HDFS.HDFSTotal),
			labelStyle.Render("Storage"),
			valueStyle.Render(
				fmt.Sprintf("%s / %s GB", node.HDFS.HDFSUsed, node.HDFS.HDFSTotal),
			),
			labelStyle.Render("Blocks"),
			// valueStyle.Render(node.Blocks),
		)

		yarn := fmt.Sprintf(
			"%s\n%s %s\n%s %s\n%s %s",
			titleStyle.Render("YARN"),
			labelStyle.Render("Status"),
			renderStatus(node.YARNStatus),
			labelStyle.Render("Memory"),
			valueStyle.Render(
				fmt.Sprintf(
					"%s / %s GB",
					node.YARNMemoryUsed,
					node.YARNMemoryTotal,
				),
			),
			labelStyle.Render("Containers"),
			valueStyle.Render(node.Containers),
		)

		details := lipgloss.JoinHorizontal(
			lipgloss.Top,
			panelStyle.Width(30).Render(docker),
			" ",
			panelStyle.Width(30).Render(hdfs),
			" ",
			panelStyle.Width(30).Render(yarn),
		)

		selected := fmt.Sprintf(
			"%s %s",
			subtitleStyle.Render("Selected node:"),
			titleStyle.Render(node.Name),
		)

		s.WriteString(selected)
		s.WriteString("\n\n")
		s.WriteString(details)
	}

	s.WriteString("\n\n")

	footer := footerStyle.Render(
		"↑/↓ navigate   •   enter details   •   r refresh   •   q quit",
	)

	s.WriteString(footer)

	return tea.NewView(s.String())
}
