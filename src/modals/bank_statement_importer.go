package modals

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"
	"time"

	"local/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ncruces/zenity"
)

// Hardcoded only ING bank, who cares about other banks frfr
// Also hardcoded: semicolon-separated values, decimal commas
type bankStatementImporter struct {
	fileLoaded bool

	activeInput int

	previewTable        table.Model
	parserPicker        itempicker.Model
	journalPicker       itempicker.Model
	accountLedgerPicker itempicker.Model
	bankLedgerPicker    itempicker.Model
}

type bankStatementParser interface {
	itempicker.Item

	compileRows(data []table.Row, accountLedger, bankLedger int) ([]database.EntryRow, error)
}

func NewBankStatementImporter() *bankStatementImporter {
	table := table.New()
	table.Focus()

	parserPicker := itempicker.New([]itempicker.Item{IngParser{}})
	journalPicker := itempicker.New(database.AvailableJournalsAsItempickerItems())
	accountLedgerPicker := itempicker.New(database.AvailableLedgersAsItempickerItems())
	bankLedgerPicker := itempicker.New(database.AvailableLedgersAsItempickerItems())

	return &bankStatementImporter{
		previewTable:        table,
		parserPicker:        parserPicker,
		journalPicker:       journalPicker,
		accountLedgerPicker: accountLedgerPicker,
		bankLedgerPicker:    bankLedgerPicker,
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
			return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
		}

		rows, columns := buildTableColumns(data)
		bsi.previewTable.SetColumns(columns)
		bsi.previewTable.SetRows(rows)

		return bsi, nil

	case meta.SwitchFocusMsg:
		numInputs := 5

		switch message.Direction {
		case meta.NEXT:
			bsi.activeInput++
			bsi.activeInput %= numInputs

		case meta.PREVIOUS:
			bsi.activeInput--

			if bsi.activeInput < 0 {
				bsi.activeInput += numInputs
			}

			// TODO: when switching to table, highlight last row

		default:
			panic(fmt.Sprintf("unexpected meta.Sequence: %#v", message.Direction))
		}

		return bsi, nil

	case tea.KeyMsg:
		switch bsi.activeInput {
		case 0:
			new, cmd := bsi.parserPicker.Update(message)
			bsi.parserPicker = new

			return bsi, cmd

		case 1:
			new, cmd := bsi.journalPicker.Update(message)
			bsi.journalPicker = new

			return bsi, cmd

		case 2:
			new, cmd := bsi.accountLedgerPicker.Update(message)
			bsi.accountLedgerPicker = new

			return bsi, cmd

		case 3:
			new, cmd := bsi.bankLedgerPicker.Update(message)
			bsi.bankLedgerPicker = new

			return bsi, cmd

		case 4:
			// Pass
			return bsi, nil

		default:
			panic(fmt.Sprintf("unexpected bsi.activeInput: %#v", bsi.activeInput))
		}

	case meta.NavigateMsg:
		if bsi.activeInput != 4 {
			return bsi, meta.MessageCmd(fmt.Errorf("jk navigation only works within the table"))
		}

		keyMsg := meta.NavigateMessageToKeyMsg(message)

		new, cmd := bsi.previewTable.Update(keyMsg)
		bsi.previewTable = new

		return bsi, cmd

	case meta.CommitMsg:
		entry := database.Entry{
			Journal: bsi.journalPicker.Value().(database.Journal).Id,
			Notes:   meta.Notes{},
		}

		rows, err := bsi.parserPicker.Value().(bankStatementParser).compileRows(
			bsi.previewTable.Rows(),
			bsi.accountLedgerPicker.Value().(database.Ledger).Id,
			bsi.bankLedgerPicker.Value().(database.Ledger).Id,
		)
		if err != nil {
			return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
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

	style := lipgloss.NewStyle()
	highlightStyle := table.DefaultStyles().Selected
	cellStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 1)

	formatSelectorStyle := style
	journalSelectorStyle := style
	accountLedgerSelectorStyle := style
	bankLedgerSelectorStyle := style
	previewTableStyles := table.DefaultStyles()
	previewTableStyles.Selected = lipgloss.NewStyle()

	switch bsi.activeInput {
	case 0:
		formatSelectorStyle = highlightStyle
	case 1:
		journalSelectorStyle = highlightStyle
	case 2:
		accountLedgerSelectorStyle = highlightStyle
	case 3:
		bankLedgerSelectorStyle = highlightStyle
	case 4:
		previewTableStyles.Selected = highlightStyle
	default:
		panic(fmt.Sprintf("unexpected bsi.activeInput: %#v", bsi.activeInput))
	}

	var result strings.Builder

	formatSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"File format",
		" ",
		formatSelectorStyle.Render(bsi.parserPicker.View()),
	))

	journalSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"Journal",
		" ",
		journalSelectorStyle.Render(bsi.journalPicker.View()),
	))

	accountLedgerSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"Accounts ledger",
		" ",
		accountLedgerSelectorStyle.Render(bsi.accountLedgerPicker.View()),
	))

	bankLedgerSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"Bank ledger",
		" ",
		bankLedgerSelectorStyle.Render(bsi.bankLedgerPicker.View()),
	))

	result.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		formatSelectorRendered,
		journalSelectorRendered,
		accountLedgerSelectorRendered,
		bankLedgerSelectorRendered,
	))

	result.WriteString("\n\n")

	bsi.previewTable.SetStyles(previewTableStyles)
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

	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})

	return meta.MotionSet{Normal: normalMotions}
}

func (bsi *bankStatementImporter) CommandSet() meta.CommandSet {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(result)
}

func (bsi *bankStatementImporter) Reload() view.View {
	return NewBankStatementImporter()
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
			colWidths[i] = min(max(colWidths[i], len(val)), 30)
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

type IngParser struct{}

func (p IngParser) compileRows(data []table.Row, accountLedger, bankLedger int) ([]database.EntryRow, error) {
	var result []database.EntryRow

	for _, row := range data {
		date, err := time.Parse("20060102", row[0])
		if err != nil {
			return nil, err
		}

		valueParts := strings.Split(row[6], ",")

		whole, err := strconv.Atoi(valueParts[0])
		if err != nil {
			return nil, err
		}
		decimal, err := strconv.Atoi(valueParts[1])
		if err != nil {
			return nil, err
		}

		value := whole*100 + decimal

		if row[5] == "Debit" {
			value *= -1
		}

		entryRow := database.EntryRow{
			Date:        database.Date(date),
			Ledger:      accountLedger,
			Account:     nil,
			Description: row[8],
			Document:    nil,
			Value:       database.CurrencyValue(value),
			Reconciled:  false,
		}
		result = append(result, entryRow)

		counterpartRow := database.EntryRow{
			Date:        database.Date(date),
			Ledger:      bankLedger,
			Account:     nil,
			Description: row[8],
			Document:    nil,
			Value:       database.CurrencyValue(-value),
			Reconciled:  false,
		}
		result = append(result, counterpartRow)
	}

	return result, nil
}

func (p IngParser) String() string {
	return "ING"
}

func (p IngParser) CompareId() int {
	return 0
}
