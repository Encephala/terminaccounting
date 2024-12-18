package main

// This file is for quick ad-hoc testing of functionality

import (
	"bubbles/itempicker"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type testModel struct {
	model itempicker.Model
}

type Item string

func (i Item) String() string {
	return string(i)
}

func main() {
	items := []itempicker.Item{
		Item("first"),
		Item("second"),
		Item("third"),
	}

	model := &testModel{
		model: itempicker.New(items),
	}

	print(fmt.Sprintf("Max view length %d\n", model.model.MaxViewLength()))

	_, err := tea.NewProgram(model).Run()

	fmt.Printf("Finished: %v\n", err)
}

func (tm *testModel) Init() tea.Cmd {
	return tm.model.Init()
}

func (tm *testModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		if message.Type == tea.KeyCtrlC {
			return tm, tea.Quit
		}
	}

	model, cmd := tm.model.Update(message)
	tm.model = model

	return tm, cmd
}

func (tm *testModel) View() string {
	return tm.model.View()
}
