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

func (bsi *bankImporter) Init() tea.Cmd {
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

func (bsi *bankImporter) Update(message tea.Msg) (view.View, tea.Cmd) {
	numInputs := 5

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		bsi.width = message.Width
		bsi.height = message.Height

		// -4 for horizontal padding on both sides
		bsi.preview.Width = message.Width - 4
		// -9 for the various inputs and confirmation prompt and vertical padding
		bsi.preview.Height = message.Height - 9

		bsi.colWidths = bsi.calculateColWidths(message.Width)

		return bsi, nil

	case meta.FileSelectedMsg:
		bsi.fileLoaded = true

		data, err := bsi.readFile(message.File)

		if err != nil {
			return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
		}

		bsi.headers = data[0]
		bsi.data = data[1:]

		bsi.colWidths = bsi.calculateColWidths(bsi.width)

		return bsi, nil

	case meta.SwitchFocusMsg:
		if bsi.activeInput == numInputs-1 {
			switch message.Direction {
			case meta.NEXT:
				if bsi.activeRow == bsi.preview.TotalLineCount()-1 {
					// Don't change table, just move focus to input 0
					break
				}

				bsi.activeRow++
				bsi.scrollViewport()

				return bsi, nil

			case meta.PREVIOUS:
				if bsi.activeRow == 0 {
					// Don't change table, just move focus to input 0
					break
				}

				bsi.activeRow--
				bsi.scrollViewport()

				return bsi, nil

			default:
				panic(fmt.Sprintf("unexpected meta.Sequence: %#v", message.Direction))
			}
		}

		switch message.Direction {
		case meta.NEXT:
			bsi.activeInput++
			bsi.activeInput %= numInputs

			if bsi.activeInput == numInputs-1 {
				bsi.activeRow = 0
				bsi.preview.GotoTop()
			}

		case meta.PREVIOUS:
			bsi.activeInput--

			if bsi.activeInput < 0 {
				bsi.activeInput += numInputs
			}

			if bsi.activeInput == numInputs-1 {
				bsi.activeRow = bsi.preview.TotalLineCount() - 1
				bsi.preview.GotoBottom()
			}

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
			new, cmd := bsi.bankLedgerPicker.Update(message)
			bsi.bankLedgerPicker = new

			return bsi, cmd

		case 3:
			// Pass
			return bsi, nil

		default:
			panic(fmt.Sprintf("unexpected bsi.activeInput: %#v", bsi.activeInput))
		}

	case meta.NavigateMsg:
		if bsi.activeInput != numInputs-1 {
			return bsi, meta.MessageCmd(errors.New("jk navigation only works within preview table"))
		}

		switch message.Direction {
		case meta.DOWN:
			if bsi.activeRow < bsi.preview.TotalLineCount()-1 {
				bsi.activeRow++
			}
		case meta.UP:
			if bsi.activeRow > 0 {
				bsi.activeRow--
			}
		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

		bsi.scrollViewport()

		return bsi, nil

	case meta.JumpVerticalMsg:
		if bsi.activeInput != numInputs-1 {
			return bsi, meta.MessageCmd(errors.New("gg/G navigation only works within preview table"))
		}

		if message.ToEnd {
			bsi.activeRow = bsi.preview.TotalLineCount() - 1
			bsi.preview.GotoBottom()
		} else {
			bsi.activeRow = 0
			bsi.preview.GotoTop()
		}

		return bsi, nil

	case meta.CommitMsg:
		journal := bsi.journalPicker.Value()
		if journal == nil {
			return bsi, meta.MessageCmd(errors.New("no journal selected (none available)"))
		}

		// This assumes only a single ledger is the accounts ledger
		accountLedgerIndex := slices.IndexFunc(database.AvailableLedgers, func(ledger database.Ledger) bool {
			return ledger.IsAccounts
		})
		if accountLedgerIndex == -1 {
			panic("this was checked for before, wut")
		}
		accountLedger := database.AvailableLedgers[accountLedgerIndex]

		bankLedger := bsi.bankLedgerPicker.Value()
		if bankLedger == nil {
			return bsi, meta.MessageCmd(errors.New("no bank ledger selected (none available)"))
		}

		rows, err := bsi.parserPicker.Value().(bankParser).compileRows(
			bsi.data,
			accountLedger.Id,
			bankLedger.(database.Ledger).Id,
		)
		if err != nil {
			return bsi, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.QuitMsg{}))
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

		return bsi, meta.MessageCmd(switchViewMsg)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (bsi *bankImporter) View() string {
	// Don't render modal until we've loaded the bank file
	if !bsi.fileLoaded {
		return ""
	}

	style := lipgloss.NewStyle()
	highlightStyle := style.Foreground(lipgloss.Color("212"))
	cellStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 1)

	formatSelectorStyle := style
	journalSelectorStyle := style
	bankLedgerSelectorStyle := style

	highlightRow := false

	switch bsi.activeInput {
	case 0:
		formatSelectorStyle = highlightStyle
	case 1:
		journalSelectorStyle = highlightStyle
	case 2:
		bankLedgerSelectorStyle = highlightStyle
	case 3:
		highlightRow = true
	default:
		panic(fmt.Sprintf("unexpected bsi.activeInput: %#v", bsi.activeInput))
	}

	bsi.setViewportContent(highlightRow)

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
		bankLedgerSelectorRendered,
	))

	result.WriteString("\n\n")

	headersStyled := make([]string, len(bsi.headers))
	usedColumns := bsi.parserPicker.Value().(bankParser).usedColumns()
	for i, header := range bsi.headers {
		style := lipgloss.NewStyle()
		if slices.Contains(usedColumns, i) {
			style = style.Foreground(lipgloss.Color("212"))
		}

		headersStyled[i] = style.Render(header)
	}
	result.WriteString(bsi.renderRow(headersStyled))

	result.WriteString("\n")

	result.WriteString(bsi.preview.View())

	result.WriteString("\n\n")

	result.WriteString(lipgloss.NewStyle().Italic(true).Render("Type :write to create the entry"))

	return result.String()
}

func (bsi *bankImporter) AllowsInsertMode() bool {
	return true
}

func (bsi *bankImporter) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (bsi *bankImporter) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})

	normalMotions.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{ToEnd: false})
	normalMotions.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{ToEnd: true})

	return meta.MotionSet{Normal: normalMotions}
}

func (bsi *bankImporter) CommandSet() meta.CommandSet {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(result)
}

func (bsi *bankImporter) Reload() view.View {
	return NewBankImporter()
}

func (bsi *bankImporter) readFile(path string) ([][]string, error) {
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

func (bsi *bankImporter) calculateColWidths(totalWidth int) []int {
	if bsi.data == nil {
		return nil
	}

	numCols := len(bsi.data[0])

	colWidths := make([]int, numCols)
	for j, header := range bsi.headers {
		colWidths[j] = len(header)
	}

	for _, row := range bsi.data {
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

func (bsi *bankImporter) setViewportContent(doHighlight bool) {
	rows := []string{}

	// Build rows, skipping header row
	for i, row := range bsi.data {
		style := lipgloss.NewStyle()
		if doHighlight && i == bsi.activeRow {
			style = style.Foreground(lipgloss.Color("212"))
		}

		rows = append(rows, style.Render(bsi.renderRow(row)))
	}

	bsi.preview.SetContent(strings.Join(rows, "\n"))
}

func (bsi *bankImporter) renderRow(values []string) string {
	if len(values) != len(bsi.colWidths) {
		panic("you absolute dingus")
	}

	newStyle := lipgloss.NewStyle()

	var result strings.Builder
	for i := range values {
		style := newStyle.Width(bsi.colWidths[i])
		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(ansi.Truncate(values[i], bsi.colWidths[i], "…")))
	}

	return result.String()
}

func (bsi *bankImporter) scrollViewport() {
	if bsi.activeRow >= bsi.preview.YOffset+bsi.preview.Height {
		bsi.preview.ScrollDown(bsi.activeRow - bsi.preview.YOffset - bsi.preview.Height + 1)
	}

	if bsi.activeRow < bsi.preview.YOffset {
		bsi.preview.ScrollUp(bsi.preview.YOffset - bsi.activeRow)
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

		counterpartyAccount := row[3]
		var matchedAccountId *int
		indexMatchedAccount := slices.IndexFunc(database.AvailableAccounts, func(a database.Account) bool {
			return a.HasBankNumber(counterpartyAccount)
		})
		if indexMatchedAccount != -1 {
			matchedAccountId = &database.AvailableAccounts[indexMatchedAccount].Id
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

	indexIBAN := strings.Index(description, "IBAN:")

	result := description[indexDescription+len("Description: ") : indexIBAN]

	return &result
}
