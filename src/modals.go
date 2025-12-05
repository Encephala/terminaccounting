package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ncruces/zenity"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

func newOverlay(main *terminaccounting) *overlay.Model {
	return overlay.New(
		main.modal,
		main.appManager,
		overlay.Center,
		overlay.Center,
		0,
		1,
	)
}

type textModal struct {
	message string
}

func (tm *textModal) Init() tea.Cmd {
	return nil
}

func (tm *textModal) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return tm, nil
}

func (tm *textModal) View() string {
	return meta.ModalStyle.Render(tm.message)
}

func (tm *textModal) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (tm *textModal) MotionSet() meta.MotionSet {
	return meta.MotionSet{}
}

func (tm *textModal) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

// Hardcoded only ING bank, who cares about other banks frfr
// Also hardcoded: semicolon-separated values, decimal commas
type bankStatementImporter struct {
	fileLoaded bool

	previewTable table.Model
}

func newBankStatementImporter() *bankStatementImporter {
	table := table.New()
	table.Focus()

	return &bankStatementImporter{
		previewTable: table,
	}
}

func (bsi *bankStatementImporter) Init() tea.Cmd {
	return func() tea.Msg {
		file, err := zenity.SelectFile(
			zenity.Title("Select bank file to import"),
			zenity.FileFilter{
				Patterns: []string{"*.csv"},
			},
		)

		if err != nil {
			return tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
		}

		return meta.FileSelectedMsg{
			File: file,
		}
	}
}

func (bsi *bankStatementImporter) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		return bsi, nil

	case meta.FileSelectedMsg:
		bsi.fileLoaded = true

		data, err := bsi.readFile(message.File)

		if err != nil {
			return nil, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
		}

		rows, columns := buildTableColumns(data)
		bsi.previewTable.SetColumns(columns)
		bsi.previewTable.SetRows(rows)

		return bsi, nil

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

		new, cmd := bsi.previewTable.Update(keyMsg)
		bsi.previewTable = new

		return bsi, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))

	}
}

func (bsi *bankStatementImporter) View() string {
	// Don't render modal until we've loaded the bank file
	if !bsi.fileLoaded {
		return ""
	}

	return meta.ModalStyle.Render(bsi.previewTable.View())
}

func (bsi *bankStatementImporter) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (bsi *bankStatementImporter) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	return meta.MotionSet{Normal: normalMotions}
}

func (bsi *bankStatementImporter) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

func (bsi *bankStatementImporter) readFile(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	result, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func buildTableColumns(data [][]string) ([]table.Row, []table.Column) {
	colNames := data[0]

	colWidths := make([]int, len(colNames))
	rows := []table.Row{}

	for i, row := range data {
		// Set column width to widest value in column, up to a maximum width
		for i, val := range row {
			colWidths[i] = min(max(colWidths[i], len(val)), 20)
		}

		if i > 0 {
			rows = append(rows, row)
		}
	}

	var columns []table.Column
	for i := range colNames {
		columns = append(columns, table.Column{
			Title: colNames[i],
			Width: colWidths[i],
		})
	}

	return rows, columns
}
