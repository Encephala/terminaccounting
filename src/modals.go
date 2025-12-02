package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
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

func (mm textModal) Init() tea.Cmd {
	return nil
}

func (mm textModal) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return mm, nil
}

func (mm textModal) View() string {
	return meta.ModalStyle.Render(mm.message)
}

// Hardcoded only ING bank, who cares about other banks frfr
// Also hardcoded: semicolon-separated values, decimal commas
type bankStatementImporter struct {
	filepicker filepicker.Model

	state bankStatementImporterState

	previewTable table.Model
}

type bankStatementImporterState string

const PICKING_STATE bankStatementImporterState = "PICKING"
const READ_AND_CONFIRM_STATE bankStatementImporterState = "READ, WAIT FOR CONFIRM"

func newBankStatementImporter() *bankStatementImporter {
	filepicker := filepicker.New()
	filepicker.SetHeight(8)
	filepicker.CurrentDirectory, _ = os.Getwd()
	filepicker.AllowedTypes = []string{".csv"}

	table := table.New()
	table.Focus()

	return &bankStatementImporter{
		filepicker:   filepicker,
		state:        PICKING_STATE,
		previewTable: table,
	}
}

func (bsi *bankStatementImporter) Init() tea.Cmd {
	return bsi.filepicker.Init()
}

func (bsi *bankStatementImporter) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		return bsi, nil

	case meta.ReadBankFileMsg:
		rows, columns := buildTableColumns(message.Data)
		bsi.previewTable.SetColumns(columns)
		bsi.previewTable.SetRows(rows)

		return bsi, nil

	default:
		// Can't panic, because `readDirMsg` is needed but is private to the filepicker package
		// panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))

		switch bsi.state {
		case PICKING_STATE:
			new, cmd := bsi.filepicker.Update(message)
			bsi.filepicker = new

			if ok, path := bsi.filepicker.DidSelectFile(message); ok {
				cmd = tea.Batch(cmd, bsi.readFile(path))

				bsi.state = READ_AND_CONFIRM_STATE
			}

			return bsi, cmd

		case READ_AND_CONFIRM_STATE:
			if navigateMsg, ok := message.(meta.NavigateMsg); ok {
				keyMsg := meta.NavigateMessageToKeyMsg(navigateMsg)

				new, cmd := bsi.previewTable.Update(keyMsg)
				bsi.previewTable = new

				return bsi, cmd
			}

			panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))

		default:
			panic(fmt.Sprintf("unexpected bankStatementImporterState: %#v", bsi.state))
		}
	}
}

func (bsi *bankStatementImporter) View() string {
	var view string

	switch bsi.state {
	case PICKING_STATE:
		view = bsi.filepicker.View()

	case READ_AND_CONFIRM_STATE:
		view = bsi.previewTable.View()
	}

	return meta.ModalStyle.Render(view)
}

func (bsi *bankStatementImporter) readFile(path string) tea.Cmd {
	file, err := os.Open(path)
	if err != nil {
		return tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'

	result, err := reader.ReadAll()
	if err != nil {
		return tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
	}

	return meta.MessageCmd(meta.ReadBankFileMsg{
		Data: result,
	})
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
