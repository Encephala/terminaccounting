package modals

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"
	"time"

	"terminaccounting/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ncruces/zenity"
)

// Hardcoded only ING bank, who cares about other banks frfr
// Also hardcoded: semicolon-separated values, decimal commas
type bankImporter struct {
	width, height int

	fileLoaded bool

	activeInput int
	activeRow   int

	headers   []string
	data      [][]string
	colWidths []int

	preview viewport.Model

	parserPicker     itempicker.Model
	journalPicker    itempicker.Model
	bankLedgerPicker itempicker.Model
}

type bankParser interface {
	itempicker.Item

	usedColumns() []int

	compileRows(data [][]string, accountLedger, bankLedger int) ([]database.EntryRow, error)
}

func NewBankImporter() *bankImporter {
	parserPicker := itempicker.New([]itempicker.Item{ingParser{}})
	journalPicker := itempicker.New(database.AvailableJournalsAsItempickerItems())
	bankLedgerPicker := itempicker.New(database.AvailableLedgersAsItempickerItems())

	return &bankImporter{
		preview:          viewport.New(0, 0),
		parserPicker:     parserPicker,
		journalPicker:    journalPicker,
		bankLedgerPicker: bankLedgerPicker,
	}
}

func (bi *bankImporter) Init() tea.Cmd {
	// Verify that an accounts ledger is
	indexAccountsLedger := database.GetAccountsLedger()
	if indexAccountsLedger == nil {
		return tea.Batch(
			meta.MessageCmd(meta.QuitMsg{}),
			meta.MessageCmd(errors.New("no accounts ledger configured yet but is needed for import")),
		)
	}

	pickFileCmd := func() tea.Msg {
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

	return pickFileCmd
}

func (bi *bankImporter) Update(message tea.Msg) (view.View, tea.Cmd) {
	numInputs := 4

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		bi.width = message.Width
		bi.height = message.Height

		// -4 for horizontal padding on both sides
		bi.preview.Width = message.Width - 4
		// -9 for the various inputs and confirmation prompt and vertical padding
		bi.preview.Height = message.Height - 9

		bi.colWidths = bi.calculateColWidths(message.Width)

		return bi, nil

	case meta.FileSelectedMsg:
		bi.fileLoaded = true

		data, err := bi.readFile(message.File)

		if err != nil {
			return bi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
		}

		bi.headers = data[0]
		bi.data = data[1:]

		bi.colWidths = bi.calculateColWidths(bi.width)

		return bi, nil

	case meta.SwitchFocusMsg:
		if bi.activeInput == numInputs-1 {
			switch message.Direction {
			case meta.NEXT:
				if bi.activeRow == bi.preview.TotalLineCount()-1 {
					// Don't change table, just move focus to input 0
					break
				}

				bi.activeRow++
				bi.scrollViewport()

				return bi, nil

			case meta.PREVIOUS:
				if bi.activeRow == 0 {
					// Don't change table, just move focus to input 0
					break
				}

				bi.activeRow--
				bi.scrollViewport()

				return bi, nil

			default:
				panic(fmt.Sprintf("unexpected meta.Sequence: %#v", message.Direction))
			}
		}

		switch message.Direction {
		case meta.NEXT:
			bi.activeInput++
			bi.activeInput %= numInputs

			if bi.activeInput == numInputs-1 {
				bi.activeRow = 0
				bi.preview.GotoTop()
			}

		case meta.PREVIOUS:
			bi.activeInput--

			if bi.activeInput < 0 {
				bi.activeInput += numInputs
			}

			if bi.activeInput == numInputs-1 {
				bi.activeRow = bi.preview.TotalLineCount() - 1
				bi.preview.GotoBottom()
			}

		default:
			panic(fmt.Sprintf("unexpected meta.Sequence: %#v", message.Direction))
		}

		return bi, nil

	case tea.KeyMsg:
		switch bi.activeInput {
		case 0:
			new, cmd := bi.parserPicker.Update(message)
			bi.parserPicker = new

			return bi, cmd

		case 1:
			new, cmd := bi.journalPicker.Update(message)
			bi.journalPicker = new

			return bi, cmd

		case 2:
			new, cmd := bi.bankLedgerPicker.Update(message)
			bi.bankLedgerPicker = new

			return bi, cmd

		case 3:
			// Pass
			return bi, nil

		default:
			panic(fmt.Sprintf("unexpected bi.activeInput: %#v", bi.activeInput))
		}

	case meta.NavigateMsg:
		if bi.activeInput != numInputs-1 {
			return bi, meta.MessageCmd(errors.New("jk navigation only works within preview table"))
		}

		switch message.Direction {
		case meta.DOWN:
			if bi.activeRow < bi.preview.TotalLineCount()-1 {
				bi.activeRow++
			}
		case meta.UP:
			if bi.activeRow > 0 {
				bi.activeRow--
			}
		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

		bi.scrollViewport()

		return bi, nil

	case meta.JumpVerticalMsg:
		if bi.activeInput != numInputs-1 {
			return bi, meta.MessageCmd(errors.New("gg/G navigation only works within preview table"))
		}

		if message.Down {
			bi.activeRow = bi.preview.TotalLineCount() - 1
			bi.preview.GotoBottom()
		} else {
			bi.activeRow = 0
			bi.preview.GotoTop()
		}

		return bi, nil

	case meta.CommitMsg:
		journal := bi.journalPicker.Value()
		if journal == nil {
			return bi, meta.MessageCmd(errors.New("no journal selected (none available)"))
		}

		// This assumes only a single ledger is the accounts ledger
		accountsLedger := database.GetAccountsLedger()
		if accountsLedger == nil {
			panic("this was checked for before, wut")
		}

		bankLedger := bi.bankLedgerPicker.Value()
		if bankLedger == nil {
			return bi, meta.MessageCmd(errors.New("no bank ledger selected (none available)"))
		}

		rows, err := bi.parserPicker.Value().(bankParser).compileRows(
			bi.data,
			accountsLedger.Id,
			bankLedger.(database.Ledger).Id,
		)
		if err != nil {
			return bi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
		}

		entriesAppType := meta.ENTRIESAPP

		switchViewMsg := meta.SwitchAppViewMsg{
			App:      &entriesAppType,
			ViewType: meta.CREATEVIEWTYPE,
			Data: view.EntryPrefillData{
				Journal: journal.(database.Journal),
				Rows:    rows,
				Notes:   meta.Notes{fmt.Sprintf("Bank import %s", time.Now().Format("2006-01-02 15:04:05"))},
			},
		}

		return bi, meta.MessageCmd(switchViewMsg)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (bi *bankImporter) View() string {
	// Don't render modal until we've loaded the bank file
	if !bi.fileLoaded {
		return ""
	}

	style := lipgloss.NewStyle()
	highlightStyle := style.Foreground(lipgloss.ANSIColor(212))
	cellStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 1)

	formatSelectorStyle := style
	journalSelectorStyle := style
	bankLedgerSelectorStyle := style

	highlightRow := false

	switch bi.activeInput {
	case 0:
		formatSelectorStyle = highlightStyle
	case 1:
		journalSelectorStyle = highlightStyle
	case 2:
		bankLedgerSelectorStyle = highlightStyle
	case 3:
		highlightRow = true
	default:
		panic(fmt.Sprintf("unexpected bi.activeInput: %#v", bi.activeInput))
	}

	bi.setViewportContent(highlightRow)

	var result strings.Builder

	formatSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"File format",
		" ",
		formatSelectorStyle.Render(bi.parserPicker.View()),
	))

	journalSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"Journal",
		" ",
		journalSelectorStyle.Render(bi.journalPicker.View()),
	))

	bankLedgerSelectorRendered := cellStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		"Bank ledger",
		" ",
		bankLedgerSelectorStyle.Render(bi.bankLedgerPicker.View()),
	))

	result.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		formatSelectorRendered,
		journalSelectorRendered,
		bankLedgerSelectorRendered,
	))

	result.WriteString("\n\n")

	headersStyled := make([]string, len(bi.headers))
	usedColumns := bi.parserPicker.Value().(bankParser).usedColumns()
	for i, header := range bi.headers {
		style := lipgloss.NewStyle()
		if slices.Contains(usedColumns, i) {
			style = style.Foreground(lipgloss.ANSIColor(212))
		}

		headersStyled[i] = style.Render(header)
	}
	result.WriteString(bi.renderRow(headersStyled))

	result.WriteString("\n")

	result.WriteString(bi.preview.View())

	result.WriteString("\n\n")

	result.WriteString(lipgloss.NewStyle().Italic(true).Render("Type :write to create the entry"))

	return result.String()
}

func (bi *bankImporter) Title() string {
	// TODO?
	return ""
}

func (bi *bankImporter) Type() meta.ViewType {
	return meta.BANKIMPORTERVIEWTYPE
}

func (bi *bankImporter) AllowsInsertMode() bool {
	return true
}

func (bi *bankImporter) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (bi *bankImporter) MotionSet() meta.Trie[tea.Msg] {
	var motions meta.Trie[tea.Msg]

	motions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	motions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	motions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	motions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})

	motions.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{Down: false})
	motions.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{Down: true})

	return motions
}

func (bi *bankImporter) CommandSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return result
}

func (bi *bankImporter) Reload() view.View {
	return NewBankImporter()
}

func (bi *bankImporter) readFile(path string) ([][]string, error) {
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

func (bi *bankImporter) calculateColWidths(totalWidth int) []int {
	if bi.data == nil {
		return nil
	}

	numCols := len(bi.data[0])

	colWidths := make([]int, numCols)
	for j, header := range bi.headers {
		colWidths[j] = len(header)
	}

	for _, row := range bi.data {
		for j, val := range row {
			colWidths[j] = max(colWidths[j], len(val))
		}
	}

	// Make it all fit nicely
	threshold := totalWidth / numCols

	// Fit all the short columns
	remainingWidth := totalWidth
	for _, width := range colWidths {
		if width < threshold {
			remainingWidth -= width
		}
	}

	// Distribute remaining length across other columns
	otherColWidth := remainingWidth / numCols
	for j, width := range colWidths {
		if width > otherColWidth {
			colWidths[j] = otherColWidth
		}
	}

	// Use some extra space if wasted due to integer rounding
	usedWidth := 0
	for _, width := range colWidths {
		usedWidth += width
	}
	// -2 is padding
	wastedWidth := totalWidth - usedWidth - 2*(numCols-1)

	for j := range colWidths {
		colWidths[j] += wastedWidth / numCols
	}

	return colWidths
}

func (bi *bankImporter) setViewportContent(doHighlight bool) {
	rows := []string{}

	// Build rows, skipping header row
	for i, row := range bi.data {
		style := lipgloss.NewStyle()
		if doHighlight && i == bi.activeRow {
			style = style.Foreground(lipgloss.ANSIColor(212))
		}

		rows = append(rows, style.Render(bi.renderRow(row)))
	}

	bi.preview.SetContent(strings.Join(rows, "\n"))
}

func (bi *bankImporter) renderRow(values []string) string {
	if len(values) != len(bi.colWidths) {
		panic("you absolute dingus")
	}

	newStyle := lipgloss.NewStyle()

	var result strings.Builder
	for i := range values {
		style := newStyle.Width(bi.colWidths[i])
		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(ansi.Truncate(values[i], bi.colWidths[i], "…")))
	}

	return result.String()
}

func (bi *bankImporter) scrollViewport() {
	if bi.activeRow >= bi.preview.YOffset+bi.preview.Height {
		bi.preview.ScrollDown(bi.activeRow - bi.preview.YOffset - bi.preview.Height + 1)
	}

	if bi.activeRow < bi.preview.YOffset {
		bi.preview.ScrollUp(bi.preview.YOffset - bi.activeRow)
	}
}

type ingParser struct{}

func (ip ingParser) usedColumns() []int {
	return []int{0, 3, 6, 8}
}

func (ip ingParser) compileRows(data [][]string, accountLedger, bankLedger int) ([]database.EntryRow, error) {
	var result []database.EntryRow

	for _, row := range data {
		date, err := time.Parse("20060102", row[0])
		if err != nil {
			return nil, err
		}

		rowDescription := row[8]
		parsedDescription := ip.parseDescription(rowDescription)
		if parsedDescription != nil {
			rowDescription = *parsedDescription
		}

		availableAccounts := database.AvailableAccounts()

		counterpartyAccount := row[3]
		var matchedAccountId *int
		indexMatchedAccount := slices.IndexFunc(availableAccounts, func(a database.Account) bool {
			return a.HasBankNumber(counterpartyAccount)
		})
		if indexMatchedAccount != -1 {
			matchedAccountId = &availableAccounts[indexMatchedAccount].Id
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
			Account:     matchedAccountId,
			Description: rowDescription,
			Document:    nil,
			Value:       database.CurrencyValue(value),
			Reconciled:  false,
		}
		result = append(result, entryRow)

		counterpartRow := database.EntryRow{
			Date:        database.Date(date),
			Ledger:      bankLedger,
			Account:     matchedAccountId,
			Description: rowDescription,
			Document:    nil,
			Value:       database.CurrencyValue(-value),
			Reconciled:  false,
		}
		result = append(result, counterpartRow)
	}

	return result, nil
}

func (ip ingParser) String() string {
	return "ING"
}

func (ip ingParser) CompareId() int {
	return 0
}

func (p ingParser) parseDescription(description string) *string {
	indexDescription := strings.Index(description, "Description:")
	if indexDescription == -1 {
		return nil
	}

	indexIBAN := strings.Index(description, " IBAN:")

	result := description[indexDescription+len("Description: ") : indexIBAN]

	return &result
}
