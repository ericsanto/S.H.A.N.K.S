package tui

import tea "charm.land/bubbletea/v2"

func InitialModel() Model {
	return Model{
		NodesHealth: []NodeHealth{
			{Name: "worker01", DockerStatus: "UP"},
			{Name: "worker02", DockerStatus: "UP"},
			{Name: "worker03", DockerStatus: "DOWN"},
		},
		selected: 0,
	}
}
func (m Model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}
