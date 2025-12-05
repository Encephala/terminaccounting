package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	case meta.CommitMsg:
		entry := database.Entry{
			Journal: 5, // TODO: add journal selector
			Notes:   meta.Notes{},
		}

		var rows []database.EntryRow

		for _, row := range bsi.previewTable.Rows() {
			date, err := time.Parse("20060102", row[0])
			if err != nil {
				return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
			}

			// TODO: allow selecting which column is value etc?
			valueParts := strings.Split(row[6], ",")

			whole, err := strconv.Atoi(valueParts[0])
			if err != nil {
				return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
			}
			decimal, err := strconv.Atoi(valueParts[1])
			if err != nil {
				return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
			}

			value := whole * 100
			if whole >= 0 {
				value += decimal
			} else {
				value -= decimal
			}

			entryRow := database.EntryRow{
				Date:       database.Date(date),
				Ledger:     14, // TODO: what to make of this?
				Account:    nil,
				Document:   nil,
				Value:      database.CurrencyValue(value),
				Reconciled: false,
			}

			rows = append(rows, entryRow)
		}

		createdEntryId, err := entry.Insert(rows)
		if err != nil {
			return bsi, meta.MessageCmd(err)
		}

		entries := meta.ENTRIESAPP

		return bsi, tea.Batch(
			meta.MessageCmd(meta.QuitMsg{}),
			meta.MessageCmd(meta.SwitchViewMsg{
				App:      &entries,
				ViewType: meta.UPDATEVIEWTYPE,
				Data:     createdEntryId,
			}))

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))

	}
}

func (bsi *bankStatementImporter) View() string {
	// Don't render modal until we've loaded the bank file
	if !bsi.fileLoaded {
		return ""
	}

	var result strings.Builder
	result.WriteString(bsi.previewTable.View())

	result.WriteString("\n\n")

	result.WriteString(lipgloss.NewStyle().Italic(true).Render("Type :write to create the entry"))

	return meta.ModalStyle.Render(result.String())
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
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(result)
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
