package tui

import (
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "down":
			if m.selected < len(m.NodesHealth)-1 {
				m.selected++
			}

		case "up":
			if m.selected > 0 {
				m.selected--
			}

		case "q":
			return m, tea.Quit
		}

	}

	return m, nil
}
