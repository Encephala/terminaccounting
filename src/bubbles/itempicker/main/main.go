package main

// This file is for quick ad-hoc testing of functionality

import (
	"fmt"
	"terminaccounting/bubbles/itempicker"

	tea "github.com/charmbracelet/bubbletea"
)

type testModel struct {
	model itempicker.Model
}

type Item string

// func (i Item) String() string {
// 	return string(i)
// }

type testItem string

func (ti testItem) String() string {
	return string(ti)
}

// Great hash function, no?
func (ti testItem) CompareId() int {
	sum := 0

	for _, r := range ti {
		sum += int(r)
	}

	return sum
}

func main() {
	items := []itempicker.Item{
		testItem("first"),
		testItem("second"),
		testItem("third"),
	}

	model := &testModel{
		model: itempicker.New(items),
	}

	model.model.SetValue(testItem("second"))

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
