package view

import (
	"fmt"
	"local/bubbles/itempicker"
	"log/slog"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"unicode"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	JOURNALINPUT activeInput = iota
	NOTESINPUT
	ENTRYROWINPUT
)

type EntryCreateView struct {
	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *EntryRowViewManager
	activeInput      activeInput

	availableJournals []database.Journal
	availableLedgers  []database.Ledger
	availableAccounts []database.Account

	colours meta.AppColours
}

func NewEntryCreateView(colours meta.AppColours) *EntryCreateView {
	journalInput := itempicker.New([]itempicker.Item{})
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	return &EntryCreateView{
		journalInput:     journalInput,
		notesInput:       noteInput,
		activeInput:      JOURNALINPUT,
		entryRowsManager: NewEntryRowCreateViewManager(),

		colours: colours,
	}
}

type EntryRowViewManager struct {
	rows []*EntryRowCreateView

	activeInput int
}

type EntryRowCreateView struct {
	dateInput    textinput.Model
	ledgerInput  itempicker.Model
	accountInput itempicker.Model
	// TODO: documentInput as some file selector thing
	// https://github.com/charmbracelet/bubbles/tree/master/filepicker
	debitInput  textinput.Model
	creditInput textinput.Model
}

// availableAccounts should not include the nil account (and cannot because not pointer type)
func newEntryRowCreateView(startDate *database.Date, availableLedgers []database.Ledger, availableAccounts []database.Account) *EntryRowCreateView {
	dateInput := textinput.New()
	dateInput.CharLimit = 10 // YYYY-MM-DD
	dateInput.Validate = func(input string) error {
		for _, char := range input {
			if !unicode.IsDigit(char) && char != '-' {
				return fmt.Errorf("invalid character for date %q", string(char))
			}
		}

		return nil
	}
	if startDate != nil {
		dateInput.SetValue(startDate.String())
	}

	ledgersAsItems := make([]itempicker.Item, len(availableLedgers))
	for i, ledger := range availableLedgers {
		ledgersAsItems[i] = ledger
	}
	ledgerInput := itempicker.New(ledgersAsItems)

	accountsAsItems := make([]itempicker.Item, len(availableAccounts)+1)
	var nilAccount *database.Account
	accountsAsItems[0] = nilAccount
	for i, account := range availableAccounts {
		accountsAsItems[i+1] = &account
	}
	accountInput := itempicker.New(accountsAsItems)

	result := EntryRowCreateView{
		dateInput:    dateInput,
		ledgerInput:  ledgerInput,
		accountInput: accountInput,
		debitInput:   textinput.New(),
		creditInput:  textinput.New(),
	}

	return &result
}

func (cv *EntryCreateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeSelectJournalsCmd(meta.ENTRIESAPP))
	cmds = append(cmds, database.MakeSelectLedgersCmd(meta.ENTRIESAPP))
	cmds = append(cmds, database.MakeSelectAccountsCmd(meta.ENTRIESAPP))

	return tea.Batch(cmds...)
}

func (cv *EntryCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		entryJournal := cv.journalInput.Value().(database.Journal)
		entryNotes := cv.notesInput.Value()

		entryRows, err := cv.entryRowsManager.compileRows()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		newEntry := database.Entry{
			Journal: entryJournal.Id,
			Notes:   meta.CompileNotes(entryNotes),
		}

		id, err := newEntry.Insert(entryRows)
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully created Entry %q", id,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)
	}

	return entriesCreateUpdateViewUpdate(cv, message)
}

func (cv *EntryCreateView) View() string {
	return entriesCreateUpdateViewView(cv)
}

func (cv *EntryCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:  {},
		meta.ACCOUNTMODEL: {},
		meta.JOURNALMODEL: {},
	}
}

type DeleteEntryRowMsg struct{}
type CreateEntryRowMsg struct {
	after bool
}

func (cv *EntryCreateView) MotionSet() *meta.MotionSet {
	return entriesCreateUpdateViewMotionSet()
}

func (cv *EntryCreateView) CommandSet() *meta.CommandSet {
	return entriesCreateUpdateViewCommandSet()
}

func (cv *EntryCreateView) getJournalInput() *itempicker.Model {
	return &cv.journalInput
}

func (cv *EntryCreateView) getNotesInput() *textarea.Model {
	return &cv.notesInput
}

func (cv *EntryCreateView) getManager() *EntryRowViewManager {
	return cv.entryRowsManager
}

func (cv *EntryCreateView) getActiveInput() *activeInput {
	return &cv.activeInput
}

func (cv *EntryCreateView) getColours() meta.AppColours {
	return cv.colours
}

func (cv *EntryCreateView) getAvailableLedgers() []database.Ledger {
	return cv.availableLedgers
}

func (cv *EntryCreateView) getAvailableAccounts() []database.Account {
	return cv.availableAccounts
}

func (cv *EntryCreateView) setJournals(journals []database.Journal) {
	cv.availableJournals = journals

	asItems := make([]itempicker.Item, len(journals))
	for i, journal := range journals {
		asItems[i] = journal
	}

	cv.journalInput.Items = asItems
}

func (cv *EntryCreateView) setLedgers(ledgers []database.Ledger) {
	cv.availableLedgers = ledgers

	asItems := make([]itempicker.Item, len(ledgers))
	for i, ledger := range ledgers {
		asItems[i] = ledger
	}

	for _, row := range cv.entryRowsManager.rows {
		row.ledgerInput.Items = asItems
	}
}

func (cv *EntryCreateView) setAccounts(accounts []database.Account) {
	cv.availableAccounts = accounts

	asItems := make([]itempicker.Item, len(accounts)+1)
	var nilAccount *database.Account
	asItems[0] = nilAccount
	for i, account := range accounts {
		asItems[i+1] = &account
	}

	for _, row := range cv.entryRowsManager.rows {
		row.accountInput.Items = asItems
	}
}

func (cv *EntryCreateView) title() string {
	return "Creating new Entry"
}

type EntryUpdateView struct {
	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *EntryRowViewManager
	activeInput      activeInput

	modelId           int
	startingEntry     database.Entry
	startingEntryRows []database.EntryRow

	availableJournals []database.Journal
	availableLedgers  []database.Ledger
	availableAccounts []database.Account

	colours meta.AppColours
}

func NewEntryUpdateView(id int, colours meta.AppColours) *EntryUpdateView {
	journalInput := itempicker.New([]itempicker.Item{})
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &EntryUpdateView{
		journalInput:     journalInput,
		notesInput:       noteInput,
		activeInput:      JOURNALINPUT,
		entryRowsManager: NewEntryRowCreateViewManager(),

		modelId: id,

		colours: colours,
	}

	return result
}

func (uv *EntryUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeSelectJournalsCmd(meta.ENTRIESAPP))
	cmds = append(cmds, database.MakeSelectLedgersCmd(meta.ENTRIESAPP))
	cmds = append(cmds, database.MakeSelectAccountsCmd(meta.ENTRIESAPP))

	cmds = append(cmds, database.MakeSelectEntryCmd(uv.modelId))
	cmds = append(cmds, database.MakeSelectEntryRowsCmd(uv.modelId))

	return tea.Batch(cmds...)
}

func (uv *EntryUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.CommitMsg:
		entryJournal := uv.journalInput.Value().(database.Journal)
		entryNotes := uv.notesInput.Value()

		entryRows, err := uv.entryRowsManager.compileRows()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		if uv.startingEntry.Id == 0 {
			panic("Updating entry but its starting value was not set")
		}

		newEntry := database.Entry{
			Id:      uv.startingEntry.Id,
			Journal: entryJournal.Id,
			Notes:   meta.CompileNotes(entryNotes),
		}

		err = newEntry.Update(entryRows)
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Entry %q", uv.modelId,
		)})

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			// You like how I solved this race condition?
			if uv.availableLedgers == nil || uv.availableAccounts == nil || uv.availableJournals == nil {
				return uv, meta.MessageCmd(message)
			}

			entry := message.Data.(database.Entry)
			uv.startingEntry = entry

			journal, err := database.SelectJournal(entry.Journal)
			if err != nil {
				return uv, meta.MessageCmd(err)
			}
			uv.journalInput.SetValue(itempicker.Item(journal))

			uv.notesInput.SetValue(entry.Notes.Collapse())

			return uv, nil

		case meta.ENTRYROWMODEL:
			// You like how I solved this race condition?
			if uv.availableLedgers == nil || uv.availableAccounts == nil || uv.availableJournals == nil {
				return uv, meta.MessageCmd(message)
			}

			rows := message.Data.([]database.EntryRow)
			if len(rows) == 0 {
				panic(fmt.Sprintf("How did entry %d end up being empty?", uv.modelId))
			}

			uv.startingEntryRows = rows

			formRows := uv.decompileRows(rows)
			uv.getManager().rows = formRows

			return uv, nil
		}

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case JOURNALINPUT:
			for _, journal := range uv.availableJournals {
				if journal.Id == uv.startingEntry.Journal {
					uv.journalInput.SetValue(journal)
					break
				}
			}

		case NOTESINPUT:
			uv.notesInput.SetValue(uv.startingEntry.Notes.Collapse())

		case ENTRYROWINPUT:
			uv.entryRowsManager.rows = uv.decompileRows(uv.startingEntryRows)

		default:
			panic(fmt.Sprintf("unexpected view.activeInput: %#v", uv.activeInput))
		}

		return uv, nil
	}

	return entriesCreateUpdateViewUpdate(uv, message)
}

func (uv *EntryUpdateView) View() string {
	return entriesCreateUpdateViewView(uv)
}

func (uv *EntryUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:   {},
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
		meta.ACCOUNTMODEL:  {},
		meta.JOURNALMODEL:  {},
	}
}

func (uv *EntryUpdateView) MotionSet() *meta.MotionSet {
	result := entriesCreateUpdateViewMotionSet()

	result.Normal.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	result.Normal.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return result
}

func (uv *EntryUpdateView) CommandSet() *meta.CommandSet {
	return entriesCreateUpdateViewCommandSet()
}

func (uv *EntryUpdateView) getJournalInput() *itempicker.Model {
	return &uv.journalInput
}

func (uv *EntryUpdateView) getNotesInput() *textarea.Model {
	return &uv.notesInput
}

func (uv *EntryUpdateView) getManager() *EntryRowViewManager {
	return uv.entryRowsManager
}

func (uv *EntryUpdateView) getActiveInput() *activeInput {
	return &uv.activeInput
}

func (uv *EntryUpdateView) getColours() meta.AppColours {
	return uv.colours
}

func (uv *EntryUpdateView) getAvailableLedgers() []database.Ledger {
	return uv.availableLedgers
}

func (uv *EntryUpdateView) getAvailableAccounts() []database.Account {
	return uv.availableAccounts
}

func (uv *EntryUpdateView) setJournals(journals []database.Journal) {
	uv.availableJournals = journals

	asItems := make([]itempicker.Item, len(journals))
	for i, journal := range journals {
		asItems[i] = journal
	}

	uv.journalInput.Items = asItems
}

func (uv *EntryUpdateView) setLedgers(ledgers []database.Ledger) {
	uv.availableLedgers = ledgers

	asItems := make([]itempicker.Item, len(ledgers))
	for i, ledger := range ledgers {
		asItems[i] = ledger
	}

	for _, row := range uv.entryRowsManager.rows {
		row.ledgerInput.Items = asItems
	}
}

func (uv *EntryUpdateView) setAccounts(accounts []database.Account) {
	uv.availableAccounts = accounts

	asItems := make([]itempicker.Item, len(accounts)+1)
	var nilAccount *database.Account
	asItems[0] = nilAccount
	for i, account := range accounts {
		asItems[i+1] = &account
	}

	for _, row := range uv.entryRowsManager.rows {
		row.accountInput.Items = asItems
	}
}

func (uv *EntryUpdateView) title() string {
	// TODO get some name/id/whatever for the entry here?
	return fmt.Sprintf("Update Entry: %s", "TODO")
}

func NewEntryRowCreateViewManager() *EntryRowViewManager {
	rows := make([]*EntryRowCreateView, 2)

	// Prefill with two empty rows
	rows[0] = newEntryRowCreateView(database.Today(), nil, nil)
	rows[1] = newEntryRowCreateView(database.Today(), nil, nil)

	return &EntryRowViewManager{
		rows: rows,
	}
}

func (ercvm *EntryRowViewManager) update(msg tea.Msg) (*EntryRowViewManager, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		highlightRow, highlightCol := ercvm.getActiveCoords()

		row := ercvm.rows[highlightRow]
		var cmd tea.Cmd
		switch highlightCol {
		case 0:
			row.dateInput, cmd = row.dateInput.Update(msg)

			if row.dateInput.Err != nil {
				cmd = tea.Batch(cmd, meta.MessageCmd(row.dateInput.Err))

				row.dateInput.Err = nil
			}

		case 1:
			row.ledgerInput, cmd = row.ledgerInput.Update(msg)
		case 2:
			row.accountInput, cmd = row.accountInput.Update(msg)
		case 3:
			if !validateNumberInput(msg) {
				return ercvm, meta.MessageCmd(fmt.Errorf("%s is not a valid character for a number", msg))
			}
			row.debitInput, cmd = row.debitInput.Update(msg)
			if row.creditInput.Value() != "" {
				row.creditInput.SetValue("")
			}
		case 4:
			if !validateNumberInput(msg) {
				return ercvm, meta.MessageCmd(fmt.Errorf("%s is not a valid character for a number", msg))
			}
			row.creditInput, cmd = row.creditInput.Update(msg)
			if row.debitInput.Value() != "" {
				row.debitInput.SetValue("")
			}
		}

		ercvm.rows[highlightRow] = row

		return ercvm, cmd

	case meta.NavigateMsg:
		oldRow, oldCol := ercvm.getActiveCoords()

		switch msg.Direction {
		case meta.DOWN:
			if oldRow == ercvm.numRows()-1 {
				break
			}
			ercvm.setActiveCoords(oldRow+1, oldCol)

		case meta.LEFT:
			if oldCol == 0 {
				break
			}
			ercvm.setActiveCoords(oldRow, oldCol-1)

		case meta.RIGHT:
			if oldCol == ercvm.numInputsPerRow()-1 {
				break
			}
			ercvm.setActiveCoords(oldRow, oldCol+1)

		case meta.UP:
			if oldRow == 0 {
				break
			}
			ercvm.setActiveCoords(oldRow-1, oldCol)

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", msg.Direction))
		}
		return ercvm, nil

	case meta.JumpHorizontalMsg:
		oldRow, _ := ercvm.getActiveCoords()

		if msg.ToEnd {
			ercvm.setActiveCoords(oldRow, ercvm.numInputsPerRow()-1)
		} else {
			ercvm.setActiveCoords(oldRow, 0)
		}

		return ercvm, nil

	case meta.JumpVerticalMsg:
		_, oldCol := ercvm.getActiveCoords()

		if msg.ToEnd {
			ercvm.setActiveCoords(ercvm.numRows()-1, oldCol)
		} else {
			ercvm.setActiveCoords(0, oldCol)
		}

		return ercvm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", msg))
	}
}

func (ercvm *EntryRowViewManager) view(style, highlightStyle lipgloss.Style, isActive bool) string {
	// TODO?: render using the table bubble to have them fix all the alignment and stuff
	var result strings.Builder

	length := ercvm.numRows() + 1
	highlightRow, highlightCol := ercvm.getActiveCoords()

	var idCol []string = make([]string, length)
	var dateCol []string = make([]string, length)
	var ledgerCol []string = make([]string, length)
	var accountCol []string = make([]string, length)
	var debitCol []string = make([]string, length)
	var creditCol []string = make([]string, length)

	idCol[0] = "Row"
	dateCol[0] = "Date"
	ledgerCol[0] = "Ledger"
	accountCol[0] = "Account"
	debitCol[0] = "Debit"
	creditCol[0] = "Credit"

	for i, row := range ercvm.rows {
		idStyle := style
		dateStyle := style
		ledgerStyle := style
		accountStyle := style
		debitStyle := style
		creditStyle := style

		if isActive && i == highlightRow {
			switch highlightCol {
			case 0:
				dateStyle = highlightStyle
			case 1:
				ledgerStyle = highlightStyle
			case 2:
				accountStyle = highlightStyle
			case 3:
				debitStyle = highlightStyle
			case 4:
				creditStyle = highlightStyle
			default:
				panic(fmt.Sprintf("Unexpected highlighted column %d", highlightCol))
			}
		}

		idCol[i+1] = idStyle.Render(strconv.Itoa(i))
		dateCol[i+1] = dateStyle.Render(row.dateInput.View())
		ledgerCol[i+1] = ledgerStyle.Render(row.ledgerInput.View())
		accountCol[i+1] = accountStyle.Render(row.accountInput.View())
		debitCol[i+1] = debitStyle.Render(row.debitInput.View())
		creditCol[i+1] = creditStyle.Render(row.creditInput.View())
	}

	idRendered := lipgloss.JoinVertical(lipgloss.Left, idCol...)
	dateRendered := lipgloss.JoinVertical(lipgloss.Left, dateCol...)
	ledgerRendered := lipgloss.JoinVertical(lipgloss.Left, ledgerCol...)
	accountRendered := lipgloss.JoinVertical(lipgloss.Left, accountCol...)
	debitRendered := lipgloss.JoinVertical(lipgloss.Left, debitCol...)
	creditRendered := lipgloss.JoinVertical(lipgloss.Left, creditCol...)

	entryRows := style.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		idRendered,
		" ",
		dateRendered,
		" ",
		ledgerRendered,
		" ",
		accountRendered,
		" ",
		debitRendered,
		" ",
		creditRendered,
	))

	total, err := ercvm.calculateCurrentTotal()
	var totalRendered string
	red := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1)).Italic(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2))
	if err == nil {
		if total == 0 {
			totalRendered = fmt.Sprintf("Total: %s", green.Render(total.String()))
		} else {
			totalRendered = fmt.Sprintf("Total: %s", total)
		}
	} else {
		totalRendered = red.Render("error")
	}

	result.WriteString(lipgloss.JoinVertical(
		lipgloss.Right,
		entryRows,
		totalRendered,
	))

	return result.String()
}

// Converts a slice of EntryRow "forms" to a slice of EntryRow
func (ercvm *EntryRowViewManager) compileRows() ([]database.EntryRow, error) {
	result := make([]database.EntryRow, ercvm.numRows())

	total, err := ercvm.calculateCurrentTotal()
	if err != nil {
		return nil, err
	}

	if total != 0 {
		return nil, fmt.Errorf("entry has nonzero total value %s", total)
	}

	for i, formRow := range ercvm.rows {
		formLedger := formRow.ledgerInput.Value().(database.Ledger)

		formAccount := formRow.accountInput.Value().(*database.Account)
		var accountId *int
		if formAccount != nil {
			accountId = &formAccount.Id
		}

		// TODO: Validate the date thingy
		date, err := database.ToDate(formRow.dateInput.Value())
		if err != nil {
			return nil, fmt.Errorf("row %d had date %q which isn't in yyyy-MM-dd:\n%#v", i, formRow.dateInput.Value(), err)
		}

		debitValue := formRow.debitInput.Value()
		creditValue := formRow.creditInput.Value()

		// Assert not both nonempty, because the createview should automatically clear the other field
		if debitValue != "" && creditValue != "" {
			panic(fmt.Sprintf(
				"expected only one of debit and credit nonempty in row %d, but got %s and %s",
				i, debitValue, creditValue))
		}

		if debitValue == "" && creditValue == "" {
			return nil, fmt.Errorf("row %d had no value for both debit and credit", i)
		}

		var value database.CurrencyValue
		if debitValue != "" {
			debit, err := database.ParseCurrencyValue(formRow.debitInput.Value())
			if err != nil {
				return nil, err
			}
			if debit == 0 {
				return nil, fmt.Errorf("row %d had 0 as debit value, only nonzero allowed", i)
			}

			value = debit
		}
		if creditValue != "" {
			credit, err := database.ParseCurrencyValue(formRow.creditInput.Value())
			if err != nil {
				return nil, err
			}
			if credit == 0 {
				return nil, fmt.Errorf("row %d had 0 as credit value, only nonzero allowed", i)
			}

			value = -credit
		}

		result[i] = database.EntryRow{
			Entry:      -1, // Will be inserted into the struct after entry itself has been inserted into db
			Date:       date,
			Ledger:     formLedger.Id,
			Account:    accountId,
			Document:   nil, // TODO
			Value:      value,
			Reconciled: false,
		}
	}

	return result, nil
}

// Converts a slice of EntryRow to a slice of EntryRowCreateView
func (uv *EntryUpdateView) decompileRows(rows []database.EntryRow,
) []*EntryRowCreateView {
	result := make([]*EntryRowCreateView, len(rows))

	for i, row := range rows {
		var ledger database.Ledger
		for _, l := range uv.availableLedgers {
			if l.Id == row.Ledger {
				ledger = l
			}
		}
		if ledger.Id == 0 {
			panic(fmt.Sprintf("Ledger not found for %#v", row))
		}

		var account *database.Account
		if row.Account != nil {
			for _, a := range uv.availableAccounts {
				if a.Id == *row.Account {
					account = &a
				}
			}
			if account == nil {
				panic(fmt.Sprintf("Account not found for %#v", row))
			}
		}

		formRow := newEntryRowCreateView(&row.Date, uv.availableLedgers, uv.availableAccounts)

		formRow.ledgerInput.SetValue(ledger)
		formRow.accountInput.SetValue(account)
		if row.Value > 0 {
			formRow.debitInput.SetValue(row.Value.String())
		} else if row.Value < 0 {
			formRow.creditInput.SetValue((-row.Value).String())
		}

		result[i] = formRow
	}

	return result
}

func (uv *EntryUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingEntry}
	}
}

// Returns preceeded/exceeded if the move would make the active input go "out of bounds"
func (ercvm *EntryRowViewManager) switchFocus(direction meta.Sequence) (preceeded, exceeded bool) {
	oldRow, oldCol := ercvm.getActiveCoords()

	switch direction {
	case meta.PREVIOUS:
		if oldRow == 0 && oldCol == 0 {
			ercvm.rows[0].dateInput.Blur()
			return true, false
		}

		ercvm.setActiveCoords(oldRow, oldCol-1)

	case meta.NEXT:
		if oldRow == ercvm.numRows()-1 && oldCol == ercvm.numInputsPerRow()-1 {
			ercvm.rows[oldRow].creditInput.Blur()
			return false, true
		}

		ercvm.setActiveCoords(oldRow, oldCol+1)

	default:
		panic(fmt.Sprintf("unexpected meta.Sequence: %#v", direction))
	}

	return false, false
}

func (ercvm *EntryRowViewManager) calculateCurrentTotal() (database.CurrencyValue, error) {
	var total database.CurrencyValue

	for _, row := range ercvm.rows {
		if row.debitInput.Value() != "" {
			change, err := database.ParseCurrencyValue(row.debitInput.Value())
			if err != nil {
				return 0, err
			}

			total = total.Add(change)
		}
		if row.creditInput.Value() != "" {
			change, err := database.ParseCurrencyValue(row.creditInput.Value())
			if err != nil {
				return 0, err
			}

			total = total.Subtract(change)
		}
	}

	return total, nil
}

func (ercvm *EntryRowViewManager) numRows() int {
	return len(ercvm.rows)
}

func (ercvm *EntryRowViewManager) numInputs() int {
	return ercvm.numRows() * ercvm.numInputsPerRow()
}

func (ercvm *EntryRowViewManager) numInputsPerRow() int {
	return 5
}

func (ercvm *EntryRowViewManager) getActiveCoords() (row, col int) {
	inputsPerRow := ercvm.numInputsPerRow()
	return ercvm.activeInput / inputsPerRow, ercvm.activeInput % inputsPerRow
}

func (ercvm *EntryRowViewManager) focus(direction meta.Sequence) {
	numInputs := ercvm.numInputs()

	switch direction {
	case meta.PREVIOUS:
		ercvm.activeInput = numInputs - 1
		ercvm.rows[ercvm.numRows()-1].creditInput.Focus()

	case meta.NEXT:
		ercvm.activeInput = 0
		ercvm.rows[0].dateInput.Focus()
	}
}

// Ignores an input that would make the active input go "out of bounds"
func (ercvm *EntryRowViewManager) setActiveCoords(newRow, newCol int) {
	numRow := ercvm.numRows()
	numPerRow := ercvm.numInputsPerRow()

	if newCol == -1 {
		newRow -= 1
		newCol = numPerRow - 1
	} else if newCol < -1 {
		panic("What")
	} else if newCol == numPerRow {
		newRow += 1
		newCol = 0
	} else if newCol > numPerRow {
		panic("What")
	}

	if newRow == -1 {
		return
	} else if newRow < -1 {
		panic("What")
	}
	if newRow == numRow {
		return
	} else if newRow > numRow {
		panic("What")
	}

	// Blur when leaving a textinput
	oldRow, oldCol := ercvm.getActiveCoords()
	switch oldCol {
	case 0:
		ercvm.rows[oldRow].dateInput.Blur()
	case 3:
		ercvm.rows[oldRow].debitInput.Blur()
	case 4:
		ercvm.rows[oldRow].creditInput.Blur()
	}

	ercvm.activeInput = newRow*numPerRow + newCol

	switch newCol {
	case 0:
		ercvm.rows[newRow].dateInput.Focus()
	case 3:
		ercvm.rows[newRow].debitInput.Focus()
	case 4:
		ercvm.rows[newRow].creditInput.Focus()
	}
}

// Checks if input is a digit or a period.
// NOTE: don't allow -, a negative debit is just a positive credit
func validateNumberInput(msg tea.KeyMsg) bool {
	// These are (likely) control flow stuff, allow it
	if len(msg.Runes) > 1 || len(msg.Runes) == 0 {
		return true
	}

	character := msg.Runes[0]

	if unicode.IsDigit(character) {
		return true
	}

	if character == '.' {
		return true
	}

	return false
}

func (ercvm *EntryRowViewManager) deleteRow() (*EntryRowViewManager, tea.Cmd) {
	activeRow, activeCol := ercvm.getActiveCoords()

	// If trying to delete the last row in the entry
	// CBA handling weird edge cases here
	if ercvm.numRows() == 1 {
		return ercvm, meta.MessageCmd(fmt.Errorf("cannot delete the final entryrow"))
	}

	// If about to delete the bottom-most row
	newRow, newCol := activeRow, activeCol
	if activeRow == ercvm.numRows()-1 {
		newRow -= 1

		// Switch focus first to avoid index out of bounds panic when unblurring oldRow
		ercvm.setActiveCoords(newRow, newCol)

		ercvm.rows = append(ercvm.rows[:activeRow], ercvm.rows[activeRow+1:]...)
	} else {
		// Switch focus after because otherwise the to-be-deleted row gets highlighted
		ercvm.rows = append(ercvm.rows[:activeRow], ercvm.rows[activeRow+1:]...)

		ercvm.setActiveCoords(newRow, newCol)
	}

	return ercvm, nil
}

func (ercvm *EntryRowViewManager) addRow(after bool,
	availableLedgers []database.Ledger,
	availableAccounts []database.Account,
) (*EntryRowViewManager, tea.Cmd) {
	activeRow, _ := ercvm.getActiveCoords()

	var newRow *EntryRowCreateView

	// If the row that the new-row-creation was triggered from had a valid date,
	// prefill it in the new row. Otherwise, just leave new row empty
	prefillDate, parseErr := database.ToDate(ercvm.rows[activeRow].dateInput.Value())
	if parseErr == nil {
		newRow = newEntryRowCreateView(&prefillDate, availableLedgers, availableAccounts)
	} else {
		newRow = newEntryRowCreateView(nil, availableLedgers, availableAccounts)
	}

	newRows := make([]*EntryRowCreateView, 0, ercvm.numRows()+1)

	if after {
		// Insert after activeRow
		newRows = append(newRows, ercvm.rows[:activeRow+1]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ercvm.rows[activeRow+1:]...)

		ercvm.rows = newRows

		ercvm.setActiveCoords(activeRow+1, 0)
	} else {
		// Insert before activeRow
		// Blur old active input
		// Just blur them all, cba to write a switch activeCol {}
		ercvm.rows[activeRow].dateInput.Blur()
		ercvm.rows[activeRow].debitInput.Blur()
		ercvm.rows[activeRow].creditInput.Blur()
		newRows = append(newRows, ercvm.rows[:activeRow]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, ercvm.rows[activeRow:]...)

		ercvm.rows = newRows

		// Ensure new active input is focused if need be
		ercvm.setActiveCoords(activeRow, 0)
	}

	return ercvm, nil
}

type entryCreateOrUpdateView interface {
	View

	getJournalInput() *itempicker.Model
	getNotesInput() *textarea.Model
	getManager() *EntryRowViewManager

	getActiveInput() *activeInput

	getColours() meta.AppColours

	getAvailableLedgers() []database.Ledger
	getAvailableAccounts() []database.Account

	setJournals([]database.Journal)
	setLedgers([]database.Ledger)
	setAccounts([]database.Account)

	title() string
}

func entriesCreateUpdateViewUpdate(view entryCreateOrUpdateView, message tea.Msg) (tea.Model, tea.Cmd) {
	activeInput := view.getActiveInput()
	journalInput := view.getJournalInput()
	notesInput := view.getNotesInput()
	entryRowsManager := view.getManager()

	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if *activeInput == NOTESINPUT {
			notesInput.Blur()
		}

		if *activeInput != ENTRYROWINPUT {
			switch message.Direction {
			case meta.PREVIOUS:
				*activeInput--
				if *activeInput < 0 {
					*activeInput += 3
				}
			case meta.NEXT:
				*activeInput++
				*activeInput %= 3
			}

			// If it changed to entryrow input
			if *activeInput == ENTRYROWINPUT {
				entryRowsManager.focus(message.Direction)
			}
		} else {
			preceeded, exceeded := entryRowsManager.switchFocus(message.Direction)

			if exceeded {
				*activeInput = 0
			}
			if preceeded {
				*activeInput = 1
			}
		}

		if *activeInput == NOTESINPUT {
			notesInput.Focus()
		}

		return view, nil

	case meta.NavigateMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(fmt.Errorf("hjkl navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpHorizontalMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(fmt.Errorf("$/_ navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpVerticalMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(fmt.Errorf("'gg'/'G' navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.JOURNALMODEL:
			view.setJournals(message.Data.([]database.Journal))

			return view, nil

		case meta.LEDGERMODEL:
			view.setLedgers(message.Data.([]database.Ledger))

			return view, nil

		case meta.ACCOUNTMODEL:
			view.setAccounts(message.Data.([]database.Account))

			return view, nil

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

	case tea.WindowSizeMsg:
		// TODO

		return view, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch *activeInput {
		case JOURNALINPUT:
			*journalInput, cmd = journalInput.Update(message)
		case NOTESINPUT:
			*notesInput, cmd = notesInput.Update(message)
		case ENTRYROWINPUT:
			var manager *EntryRowViewManager
			manager, cmd = entryRowsManager.update(message)
			*entryRowsManager = *manager

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", *activeInput))
		}

		return view, cmd

	case DeleteEntryRowMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(fmt.Errorf("no entry row highlighted while trying to delete one"))
		}

		manager, cmd := entryRowsManager.deleteRow()
		*entryRowsManager = *manager

		return view, cmd

	case CreateEntryRowMsg:
		if *activeInput != ENTRYROWINPUT {
			return view, meta.MessageCmd(fmt.Errorf("no entry row highlighted while trying to create one"))
		}

		var cmds []tea.Cmd

		manager, cmd := entryRowsManager.addRow(message.after, view.getAvailableLedgers(), view.getAvailableAccounts())
		*entryRowsManager = *manager
		cmds = append(cmds, cmd)

		cmds = append(cmds, meta.MessageCmd(meta.SwitchModeMsg{
			InputMode: meta.INSERTMODE,
		}))

		return view, tea.Batch(cmds...)

	default:
		// TODO (waiting for https://github.com/charmbracelet/bubbles/issues/834)
		// panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
		slog.Warn(fmt.Sprintf("unexpected tea.Msg: %#v", message))
		return view, nil
	}
}

func entriesCreateUpdateViewView(view entryCreateOrUpdateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(view.getColours().Background).Padding(0, 1).Margin(0, 0, 0, 2)

	result.WriteString(titleStyle.Render(view.title()))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)
	highlightStyle := style.Foreground(view.getColours().Foreground)

	journalInput := view.getJournalInput()
	notesInput := view.getNotesInput()
	entryRowsManager := view.getManager()

	inputWidth := journalInput.MaxViewLength()

	notesInput.SetWidth(inputWidth)
	notesFocusStyle := lipgloss.NewStyle().Foreground(view.getColours().Foreground)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	nameCol := lipgloss.JoinVertical(
		lipgloss.Right,
		style.Render("Journal"),
		style.Render("Notes"),
	)

	var inputCol string
	if *view.getActiveInput() == JOURNALINPUT {
		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			highlightStyle.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(view.getJournalInput().View()),
			style.Render(notesInput.View()),
		)
	} else if *view.getActiveInput() == NOTESINPUT {
		notesInput.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(view.getColours().Foreground)

		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			style.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(journalInput.View()),
			style.Render(notesInput.View()),
		)
	} else {
		inputCol = lipgloss.JoinVertical(
			lipgloss.Left,
			style.Width(inputWidth+2).AlignHorizontal(lipgloss.Left).Render(journalInput.View()),
			style.Render(notesInput.View()),
		)
	}

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			nameCol,
			" ",
			inputCol,
		),
	))
	result.WriteString("\n\n")

	result.WriteString(style.MarginLeft(2).Render(
		entryRowsManager.view(lipgloss.NewStyle(),
			lipgloss.NewStyle().Foreground(view.getColours().Foreground),
			*view.getActiveInput() == ENTRYROWINPUT,
		),
	))

	return result.String()
}

func entriesCreateUpdateViewMotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	// Default navigation
	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	// Create/delete rows
	normalMotions.Insert(meta.Motion{"d", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "D"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"o"}, CreateEntryRowMsg{after: true})
	normalMotions.Insert(meta.Motion{"O"}, CreateEntryRowMsg{after: false})

	// hjkl navigation in entryrows
	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	// Extra horizontal navigation
	normalMotions.Insert(meta.Motion{"$"}, meta.JumpHorizontalMsg{ToEnd: true})
	normalMotions.Insert(meta.Motion{"_"}, meta.JumpHorizontalMsg{ToEnd: false})

	// Extra vertical navigation
	normalMotions.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{ToEnd: false})
	normalMotions.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{ToEnd: true})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func entriesCreateUpdateViewCommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

type EntryDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Entry

	colours meta.AppColours
}

func NewEntryDeleteView(modelId int, colours meta.AppColours) *EntryDeleteView {
	return &EntryDeleteView{
		modelId: modelId,

		colours: colours,
	}
}

func (dv *EntryDeleteView) Init() tea.Cmd {
	return database.MakeLoadEntryDetailCmd(dv.modelId)
}

func (dv *EntryDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		entry := message.Data.(database.Entry)

		dv.model = entry

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteEntry(dv.model.Id)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted entry %q", dv.modelId,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		// TODO

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *EntryDeleteView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.colours.Background).Padding(0, 1).MarginLeft(2)
	style := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder())
	rightStyle := style.Margin(0, 0, 0, 1)

	result.WriteString(titleStyle.Render(fmt.Sprintf("Delete Entry: %d", dv.model.Id)))
	result.WriteString("\n\n")

	journalRendered := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Name"),
		rightStyle.Render(strconv.Itoa(dv.model.Journal)),
	)

	notesRendered := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Notes"),
		rightStyle.Render(strings.Join(dv.model.Notes, "\n")),
	)

	var confirmRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Italic(true).Render("Run the `:w` command to confirm"),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			journalRendered,
			notesRendered,
			"",
			confirmRow,
		),
	))

	return result.String()
}

func (dv *EntryDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL: {},
	}
}

func (dv *EntryDeleteView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *EntryDeleteView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

func (dv *EntryDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
