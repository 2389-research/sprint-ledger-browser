// ABOUTME: Root Bubbletea model for the sprint-ledger TUI.
// ABOUTME: Manages split-pane layout with table (top) and detail viewport (bottom).
package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusTable focus = iota
	focusDetail
)

type model struct {
	sprints     []Sprint
	projectName string
	table       table.Model
	viewport    viewport.Model
	focus       focus
	width       int
	height      int
	ready       bool
}

func newModel(sprints []Sprint, projectName string) model {
	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Title", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Updated", Width: 8},
	}

	rawRows := SprintsToRows(sprints)
	rows := make([]table.Row, len(rawRows))
	for i, r := range rawRows {
		rows[i] = table.Row(r)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(5),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		sprints:     sprints,
		projectName: projectName,
		table:       t,
		focus:       focusTable,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 1
		helpHeight := 1
		borderHeight := 4 // 2 lines per bordered pane × 2 panes
		availableHeight := m.height - headerHeight - helpHeight - borderHeight

		tableHeight := availableHeight * 40 / 100
		if tableHeight < 3 {
			tableHeight = 3
		}
		detailHeight := availableHeight - tableHeight
		if detailHeight < 1 {
			detailHeight = 1
		}

		contentWidth := m.width - 2
		if contentWidth < 1 {
			contentWidth = 1
		}

		m.table.SetWidth(contentWidth)
		m.table.SetHeight(tableHeight)

		if !m.ready {
			m.viewport = viewport.New(contentWidth, detailHeight)
			if len(m.sprints) > 0 {
				m.viewport.SetContent(LoadDetailContent(m.sprints[0]))
			} else {
				m.viewport.SetContent("No sprints found")
			}
			m.ready = true
		} else {
			m.viewport.Width = contentWidth
			m.viewport.Height = detailHeight
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab", "enter":
			if m.focus == focusTable {
				m.focus = focusDetail
				m.table.Blur()
			} else {
				m.focus = focusTable
				m.table.Focus()
			}
			return m, nil

		case "esc":
			if m.focus == focusDetail {
				m.focus = focusTable
				m.table.Focus()
			}
			return m, nil
		}
	}

	prevCursor := m.table.Cursor()

	if m.focus == focusTable {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)

		if m.table.Cursor() != prevCursor && m.table.Cursor() < len(m.sprints) {
			m.viewport.SetContent(LoadDetailContent(m.sprints[m.table.Cursor()]))
			m.viewport.GotoTop()
		}
	} else {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "loading..."
	}

	header := headerStyle.Width(m.width).Render(
		fmt.Sprintf("sprint-ledger: %s", m.projectName),
	)

	var tableView, detailView string
	if m.focus == focusTable {
		tableView = focusedBorder.Width(m.width - 2).Render(m.table.View())
		detailView = blurredBorder.Width(m.width - 2).Render(m.viewport.View())
	} else {
		tableView = blurredBorder.Width(m.width - 2).Render(m.table.View())
		detailView = focusedBorder.Width(m.width - 2).Render(m.viewport.View())
	}

	help := helpStyle.Render("j/k navigate • tab focus • esc back • q quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tableView,
		detailView,
		help,
	)
}
