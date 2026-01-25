package view

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/database"
	"terminaccounting/meta"
	"unicode"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type entryDetailView struct {
	DB *sqlx.DB

	// The entries whose rows are being shown
	modelId int
	model   database.Entry

	viewer *entryRowViewer
}

func NewEntriesDetailView(DB *sqlx.DB, modelId int) *entryDetailView {
	return &entryDetailView{
		DB: DB,

		modelId: modelId,

		viewer: newEntryRowViewer(meta.ENTRIESCOLOUR),
	}
}

func (dv *entryDetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeLoadEntriesDetailCmd(dv.DB, dv.modelId))
	cmds = append(cmds, database.MakeLoadEntriesRowsCmd(dv.DB, dv.modelId))

	return tea.Batch(cmds...)
}

func (dv *entryDetailView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			dv.model = message.Data.(database.Entry)

			return dv, nil

		case meta.ENTRYROWMODEL:
			return genericDetailViewUpdate(dv, message)

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}
	}

	return genericDetailViewUpdate(dv, message)
}

func (dv *entryDetailView) View() string {
	return genericDetailViewView(dv)
}

func (dv *entryDetailView) Type() meta.ViewType {
	return meta.DETAILVIEWTYPE
}

func (dv *entryDetailView) getDB() *sqlx.DB {
	return dv.DB
}

func (dv *entryDetailView) title() string {
	return fmt.Sprintf("Entry %d details", dv.model.Id)
}

func (dv *entryDetailView) getCanReconcile() bool {
	return false
}

func (dv *entryDetailView) AllowsInsertMode() bool {
	return false
}

func (dv *entryDetailView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
	}
}

func (dv *entryDetailView) MotionSet() meta.MotionSet {
	result := genericDetailViewMotionSet()

	result.Normal.Insert(meta.Motion{"g", "x"}, meta.SwitchAppViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})
	result.Normal.Insert(meta.Motion{"g", "e"}, meta.SwitchAppViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	return result
}

func (dv *entryDetailView) CommandSet() meta.CommandSet {
	return genericDetailViewCommandSet()
}

func (dv *entryDetailView) Reload() View {
	return NewEntriesDetailView(dv.DB, dv.modelId)
}

func (dv *entryDetailView) getViewer() *entryRowViewer {
	return dv.viewer
}

func (dv *entryDetailView) getColour() lipgloss.Color {
	return meta.ENTRIESCOLOUR
}

// NOTE: entries doesn't use the genericMutateView, because with the row creating it's too idiosyncratic

const (
	ENTRIESJOURNALINPUT int = iota
	ENTRIESNOTESINPUT
	ENTRIESROWINPUT
)

type entryCreateView struct {
	DB *sqlx.DB

	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *rowsMutateManager
	activeInput      int

	colour lipgloss.Color
}

func NewEntryCreateView(DB *sqlx.DB) *entryCreateView {
	journalInput := itempicker.New(database.AvailableJournalsAsItempickerItems())

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.ENTRIESCOLOUR)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	result := &entryCreateView{
		DB: DB,

		journalInput:     journalInput,
		notesInput:       notesInput,
		activeInput:      ENTRIESJOURNALINPUT,
		entryRowsManager: newRowsMutateManager(),

		colour: meta.ENTRIESCOLOUR,
	}

	return result
}

type EntryPrefillData struct {
	Journal database.Journal
	Rows    []database.EntryRow
	Notes   meta.Notes
}

// Make an EntryCreateView with the provided journal, rows prefilled into forms
func NewEntryCreateViewPrefilled(DB *sqlx.DB, data EntryPrefillData) (*entryCreateView, error) {
	result := NewEntryCreateView(DB)

	result.journalInput.SetValue(data.Journal)
	result.notesInput.SetValue(data.Notes.Collapse())

	entryRowCreateView, err := decompileRows(data.Rows)
	if err != nil {
		return nil, err
	}

	result.entryRowsManager.rows = entryRowCreateView

	return result, nil
}

type rowCreator struct {
	width int

	dateInput        textinput.Model
	ledgerInput      itempicker.Model
	accountInput     itempicker.Model
	descriptionInput textinput.Model
	// TODO: documentInput as some file selector thing
	// https://github.com/charmbracelet/bubbles/tree/master/filepicker
	debitInput  textinput.Model
	creditInput textinput.Model
}

func newRowCreator(startDate *database.Date) *rowCreator {
	dateInput := textinput.New()
	dateInput.Cursor.SetMode(cursor.CursorStatic)
	dateInput.Placeholder = "yyyy-MM-dd"
	dateInput.CharLimit = 10
	dateInput.Width = 10
	if startDate != nil {
		dateInput.SetValue(startDate.String())
	}

	ledgerInput := itempicker.New(database.AvailableLedgersAsItempickerItems())
	accountInput := itempicker.New(database.AvailableAccountsAsItempickerItems())

	descriptionInput := textinput.New()
	descriptionInput.Cursor.SetMode(cursor.CursorStatic)
	debitInput := textinput.New()
	debitInput.Cursor.SetMode(cursor.CursorStatic)
	creditInput := textinput.New()
	creditInput.Cursor.SetMode(cursor.CursorStatic)

	result := rowCreator{
		dateInput:        dateInput,
		ledgerInput:      ledgerInput,
		accountInput:     accountInput,
		descriptionInput: descriptionInput,
		debitInput:       debitInput,
		creditInput:      creditInput,
	}

	return &result
}

func (cv *entryCreateView) Init() tea.Cmd {
	return nil
}

func (cv *entryCreateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		entryJournal := cv.journalInput.Value()
		if entryJournal == nil {
			return cv, meta.MessageCmd(errors.New("no journal selected (none available)"))
		}
		entryNotes := cv.notesInput.Value()

		entryRows, err := cv.entryRowsManager.compileRows()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		newEntry := database.Entry{
			Journal: entryJournal.(database.Journal).Id,
			Notes:   meta.CompileNotes(entryNotes),
		}

		id, err := newEntry.Insert(cv.DB, entryRows)
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully created Entry \"%d\"", id,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)
	}

	return entriesMutateViewUpdate(cv, message)
}

func (cv *entryCreateView) View() string {
	return entriesMutateViewView(cv)
}

func (cv *entryCreateView) Type() meta.ViewType {
	return meta.CREATEVIEWTYPE
}

func (cv *entryCreateView) AllowsInsertMode() bool {
	return true
}

func (cv *entryCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:  {},
		meta.ACCOUNTMODEL: {},
		meta.JOURNALMODEL: {},
	}
}

type DeleteEntryRowMsg struct{}
type CreateEntryRowMsg struct {
	After bool
}

func (cv *entryCreateView) MotionSet() meta.MotionSet {
	return entriesMutateViewMotionSet()
}

func (cv *entryCreateView) CommandSet() meta.CommandSet {
	return entriesMutateViewCommandSet()
}

func (cv *entryCreateView) Reload() View {
	return NewEntryCreateView(cv.DB)
}

func (cv *entryCreateView) getJournalInput() *itempicker.Model {
	return &cv.journalInput
}

func (cv *entryCreateView) getNotesInput() *textarea.Model {
	return &cv.notesInput
}

func (cv *entryCreateView) getManager() *rowsMutateManager {
	return cv.entryRowsManager
}

func (cv *entryCreateView) getActiveInput() *int {
	return &cv.activeInput
}

func (cv *entryCreateView) getColour() lipgloss.Color {
	return cv.colour
}

func (cv *entryCreateView) title() string {
	return "Creating new Entry"
}

type entryUpdateView struct {
	DB *sqlx.DB

	journalInput     itempicker.Model
	notesInput       textarea.Model
	entryRowsManager *rowsMutateManager
	activeInput      int

	modelId           int
	startingEntry     database.Entry
	startingEntryRows []database.EntryRow

	colour lipgloss.Color
}

func NewEntryUpdateView(DB *sqlx.DB, modelId int) *entryUpdateView {
	journalInput := itempicker.New(database.AvailableJournalsAsItempickerItems())

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.ENTRIESCOLOUR)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	result := &entryUpdateView{
		DB: DB,

		journalInput:     journalInput,
		notesInput:       notesInput,
		activeInput:      ENTRIESJOURNALINPUT,
		entryRowsManager: newRowsMutateManager(),

		modelId: modelId,

		colour: meta.ENTRIESCOLOUR,
	}

	return result
}

func (uv *entryUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeSelectEntryCmd(uv.DB, uv.modelId))
	cmds = append(cmds, database.MakeSelectEntryRowsCmd(uv.DB, uv.modelId))

	return tea.Batch(cmds...)
}

func (uv *entryUpdateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.CommitMsg:
		entryJournal := uv.journalInput.Value()
		if entryJournal == nil {
			return uv, meta.MessageCmd(errors.New("no journal selected (none available)"))
		}
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
			Journal: entryJournal.(database.Journal).Id,
			Notes:   meta.CompileNotes(entryNotes),
		}

		err = newEntry.Update(uv.DB, entryRows)
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Entry \"%d\"", uv.modelId,
		)})

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			// NOTE: I assume a valid state of the database cache (ledgers/accounts/journals)

			entry := message.Data.(database.Entry)
			uv.startingEntry = entry

			journal, err := database.SelectJournal(uv.DB, entry.Journal)
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			err = uv.journalInput.SetValue(itempicker.Item(journal))
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			uv.notesInput.SetValue(entry.Notes.Collapse())

			return uv, nil

		case meta.ENTRYROWMODEL:
			// NOTE: I assume a valid state of the database cache (ledgers/accounts/journals)

			rows := message.Data.([]database.EntryRow)
			if len(rows) == 0 {
				panic(fmt.Sprintf("How did entry %d end up being empty?", uv.modelId))
			}

			uv.startingEntryRows = rows

			formRows, err := decompileRows(rows)
			if err != nil {
				return uv, meta.MessageCmd(err)
			}

			uv.getManager().rows = formRows

			return uv, nil
		}

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case ENTRIESJOURNALINPUT:
			availableJournals := database.AvailableJournals()
			availableJournalIndex := slices.IndexFunc(availableJournals, func(journal database.Journal) bool {
				return journal.Id == uv.startingEntry.Journal
			})

			if availableJournalIndex == -1 {
				panic("This won't happen, surely")
			}

			err := uv.journalInput.SetValue(availableJournals[availableJournalIndex])
			if err != nil {
				panic("This can't happen")
			}

		case ENTRIESNOTESINPUT:
			uv.notesInput.SetValue(uv.startingEntry.Notes.Collapse())

		case ENTRIESROWINPUT:
			var err error
			uv.entryRowsManager.rows, err = decompileRows(uv.startingEntryRows)

			if err != nil {
				return uv, meta.MessageCmd(err)
			}

		default:
			panic(fmt.Sprintf("unexpected view.activeInput: %#v", uv.activeInput))
		}

		return uv, nil
	}

	return entriesMutateViewUpdate(uv, message)
}

func (uv *entryUpdateView) View() string {
	return entriesMutateViewView(uv)
}

func (uv *entryUpdateView) Type() meta.ViewType {
	return meta.UPDATEVIEWTYPE
}

func (uv *entryUpdateView) AllowsInsertMode() bool {
	return true
}

func (uv *entryUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:   {},
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
		meta.ACCOUNTMODEL:  {},
		meta.JOURNALMODEL:  {},
	}
}

func (uv *entryUpdateView) MotionSet() meta.MotionSet {
	result := entriesMutateViewMotionSet()

	result.Normal.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	result.Normal.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return result
}

func (uv *entryUpdateView) CommandSet() meta.CommandSet {
	return entriesMutateViewCommandSet()
}

func (uv *entryUpdateView) Reload() View {
	return NewEntryUpdateView(uv.DB, uv.modelId)
}

func (uv *entryUpdateView) getJournalInput() *itempicker.Model {
	return &uv.journalInput
}

func (uv *entryUpdateView) getNotesInput() *textarea.Model {
	return &uv.notesInput
}

func (uv *entryUpdateView) getManager() *rowsMutateManager {
	return uv.entryRowsManager
}

func (uv *entryUpdateView) getActiveInput() *int {
	return &uv.activeInput
}

func (uv *entryUpdateView) getColour() lipgloss.Color {
	return uv.colour
}

func (uv *entryUpdateView) title() string {
	// TODO get some name/id/whatever for the entry here?
	return fmt.Sprintf("Update Entry: %s", "TODO")
}

type rowsMutateManager struct {
	width, height int

	headers []string
	rows    []*rowCreator

	isActive    bool
	activeInput int

	colWidths []int

	viewport viewport.Model
}

func newRowsMutateManager() *rowsMutateManager {
	// Prefill with two empty rows
	rows := make([]*rowCreator, 2)

	rows[0] = newRowCreator(database.Today())
	rows[1] = newRowCreator(database.Today())

	result := &rowsMutateManager{
		headers: []string{"Row", "Date", "Ledger", "Account", "Description", "Debit", "Credit"},
		rows:    rows,

		colWidths: []int{0, 0, 0, 0, 0, 0, 0},
		viewport:  viewport.New(0, 0),
	}

	return result
}

func (rmm *rowsMutateManager) Update(message tea.Msg) (*rowsMutateManager, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		rmm.width = message.Width
		rmm.height = message.Height

		// -8 for border and horizontal padding and margin
		rmm.viewport.Width = message.Width - 8
		// -6 for borders, header row, vertical margin and total row
		rmm.viewport.Height = message.Height - 6

		rmm.calculateColumnWidths()

		for _, row := range rmm.rows {
			row.width = message.Width
		}

		return rmm, nil

	case tea.KeyMsg:
		highlightRow, highlightCol := rmm.getActiveCoords()

		row := rmm.rows[highlightRow]
		var cmd tea.Cmd
		switch highlightCol {
		case 0:
			if !validateDateInput(message) {
				return rmm, meta.MessageCmd(fmt.Errorf("%q is not a valid character for a date", message))
			}
			row.dateInput, cmd = row.dateInput.Update(message)

		case 1:
			row.ledgerInput, cmd = row.ledgerInput.Update(message)
		case 2:
			row.accountInput, cmd = row.accountInput.Update(message)
		case 3:
			row.descriptionInput, cmd = row.descriptionInput.Update(message)
		case 4:
			if !validateNumberInput(message) {
				return rmm, meta.MessageCmd(fmt.Errorf("%q is not a valid character for a number", message))
			}
			row.debitInput, cmd = row.debitInput.Update(message)
			if row.creditInput.Value() != "" {
				row.creditInput.SetValue("")
			}
		case 5:
			if !validateNumberInput(message) {
				return rmm, meta.MessageCmd(fmt.Errorf("%q is not a valid character for a number", message))
			}
			row.creditInput, cmd = row.creditInput.Update(message)
			if row.debitInput.Value() != "" {
				row.debitInput.SetValue("")
			}
		}

		rmm.rows[highlightRow] = row

		return rmm, cmd

	case meta.NavigateMsg:
		oldRow, oldCol := rmm.getActiveCoords()

		switch message.Direction {
		case meta.LEFT:
			if oldCol == 0 {
				break
			}
			rmm.setActiveCoords(oldRow, oldCol-1)

		case meta.DOWN:
			if oldRow == rmm.numRows()-1 {
				break
			}
			rmm.setActiveCoords(oldRow+1, oldCol)

		case meta.UP:
			if oldRow == 0 {
				break
			}
			rmm.setActiveCoords(oldRow-1, oldCol)

		case meta.RIGHT:
			if oldCol == rmm.numInputsPerRow()-1 {
				break
			}
			rmm.setActiveCoords(oldRow, oldCol+1)

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

		return rmm, nil

	case meta.JumpHorizontalMsg:
		oldRow, _ := rmm.getActiveCoords()

		if message.ToEnd {
			rmm.setActiveCoords(oldRow, rmm.numInputsPerRow()-1)
		} else {
			rmm.setActiveCoords(oldRow, 0)
		}

		return rmm, nil

	case meta.JumpVerticalMsg:
		_, oldCol := rmm.getActiveCoords()

		if message.ToEnd {
			rmm.setActiveCoords(rmm.numRows()-1, oldCol)
		} else {
			rmm.setActiveCoords(0, oldCol)
		}

		return rmm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (rmm *rowsMutateManager) View() string {
	var result strings.Builder

	result.WriteString(rmm.renderRow(rmm.headers, nil))

	result.WriteString("\n")

	rmm.updateViewportContent()
	result.WriteString(rmm.viewport.View())

	total, err := rmm.calculateCurrentTotal()
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

	result.WriteString("\n")

	result.WriteString(totalRendered)

	return result.String()
}

func (rmm *rowsMutateManager) calculateColumnWidths() {
	idxWidth := 3
	// 10 for yyyy-MM-dd and 2 for prompt and 1 for cursor
	dateWidth := 13

	// -12 for padding between columns 6x, -4 for borders and left/right margin
	remainingWidth := rmm.width - idxWidth - dateWidth - 12 - 6
	descriptionWidth := remainingWidth / 3
	othersWidth := (remainingWidth - descriptionWidth) / 4

	rmm.colWidths = []int{idxWidth, dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth}
}

func (rmm *rowsMutateManager) renderRow(values []string, highlightedCol *int) string {
	if len(values) != len(rmm.colWidths) {
		panic("you absolute dingus")
	}

	var result strings.Builder
	for i := range values {
		style := lipgloss.NewStyle().Width(rmm.colWidths[i])

		// +1 because never highlight idx column
		if highlightedCol != nil && i == *highlightedCol+1 {
			style = style.Foreground(meta.ENTRIESCOLOUR)
		}

		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(values[i]))
	}

	return result.String()
}

func (rmm *rowsMutateManager) updateViewportContent() {
	var rows []string

	activeRow, activeCol := rmm.getActiveCoords()

	for i, row := range rmm.makeShownRows() {
		if rmm.isActive && i == activeRow {
			rows = append(rows, rmm.renderRow(row, &activeCol))
		} else {
			rows = append(rows, rmm.renderRow(row, nil))
		}
	}

	rmm.viewport.SetContent(strings.Join(rows, "\n"))
	rmm.scrollViewport()
}

func (rmm *rowsMutateManager) makeShownRows() [][]string {
	var result [][]string

	for i, row := range rmm.rows {
		var currentRow []string

		currentRow = append(currentRow, strconv.Itoa(i))
		currentRow = append(currentRow, row.dateInput.View())
		currentRow = append(currentRow, row.ledgerInput.View())
		currentRow = append(currentRow, row.accountInput.View())
		currentRow = append(currentRow, row.descriptionInput.View())
		currentRow = append(currentRow, row.debitInput.View())
		currentRow = append(currentRow, row.creditInput.View())

		result = append(result, currentRow)
	}

	return result
}

func (rmm *rowsMutateManager) scrollViewport() {
	shownRows := rmm.makeShownRows()

	// If there are fewer rows shown than fit on the viewport, show the last set of rows
	if rmm.viewport.YOffset+rmm.viewport.Height > len(shownRows) {
		rmm.viewport.ScrollUp(rmm.viewport.YOffset + rmm.viewport.Height - len(shownRows))
		return
	}

	activeRow, _ := rmm.getActiveCoords()

	if activeRow >= rmm.viewport.YOffset+rmm.viewport.Height {
		rmm.viewport.ScrollDown(activeRow - rmm.viewport.YOffset - rmm.viewport.Height + 1)
		return
	}

	if activeRow < rmm.viewport.YOffset {
		rmm.viewport.ScrollUp(rmm.viewport.YOffset - activeRow)
		return
	}
}

// Converts a slice of EntryRow "forms" to a slice of EntryRow
func (rmm *rowsMutateManager) compileRows() ([]database.EntryRow, error) {
	result := make([]database.EntryRow, rmm.numRows())

	total, err := rmm.calculateCurrentTotal()
	if err != nil {
		return nil, err
	}

	if total != 0 {
		return nil, fmt.Errorf("entry has nonzero total value %s", total)
	}

	for i, formRow := range rmm.rows {
		formLedger := formRow.ledgerInput.Value()
		if formLedger == nil {
			return nil, fmt.Errorf("invalid ledger selected in row %d (none available)", i)
		}

		formAccount := formRow.accountInput.Value()
		if formAccount == nil {
			return nil, fmt.Errorf("invalid account selected in row %d (none available)", i)
		}
		var accountId *int
		if formAccount.(*database.Account) != nil {
			accountId = &formAccount.(*database.Account).Id
		}

		// TODO: Validate the date thingy
		date, err := database.ToDate(formRow.dateInput.Value())
		if err != nil {
			return nil, fmt.Errorf("row %d had date %q which isn't in yyyy-MM-dd:\n%#v", i, formRow.dateInput.Value(), err)
		}

		formDescription := formRow.descriptionInput.Value()
		formDebit := formRow.debitInput.Value()
		formCredit := formRow.creditInput.Value()

		// Assert not both nonempty, because the createview should automatically clear the other field
		if formDebit != "" && formCredit != "" {
			panic(fmt.Sprintf(
				"expected only one of debit and credit nonempty in row %d, but got %q and %q",
				i, formDebit, formCredit))
		}

		if formDebit == "" && formCredit == "" {
			return nil, fmt.Errorf("row %d had no value for both debit and credit", i)
		}

		var value database.CurrencyValue
		if formDebit != "" {
			debit, err := database.ParseCurrencyValue(formRow.debitInput.Value())
			if err != nil {
				return nil, err
			}
			if debit == 0 {
				return nil, fmt.Errorf("row %d had 0 as debit value, only nonzero allowed", i)
			}

			value = debit
		}
		if formCredit != "" {
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
			Entry:       -1, // Will be inserted into the struct after entry itself has been inserted into db
			Date:        date,
			Ledger:      formLedger.(database.Ledger).Id,
			Account:     accountId,
			Description: formDescription,
			Document:    nil, // TODO
			Value:       value,
			Reconciled:  false,
		}
	}

	return result, nil
}

func (uv *entryUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingEntry}
	}
}

// Returns preceeded/exceeded if the move would make the active input go "out of bounds"
func (rmm *rowsMutateManager) switchFocus(direction meta.Sequence) (preceeded, exceeded bool) {
	oldRow, oldCol := rmm.getActiveCoords()

	switch direction {
	case meta.PREVIOUS:
		if oldRow == 0 && oldCol == 0 {
			rmm.rows[0].dateInput.Blur()
			rmm.isActive = false
			return true, false
		}

		rmm.setActiveCoords(oldRow, oldCol-1)

	case meta.NEXT:
		if oldRow == rmm.numRows()-1 && oldCol == rmm.numInputsPerRow()-1 {
			rmm.rows[oldRow].creditInput.Blur()
			rmm.isActive = false
			return false, true
		}

		rmm.setActiveCoords(oldRow, oldCol+1)

	default:
		panic(fmt.Sprintf("unexpected meta.Sequence: %#v", direction))
	}

	return false, false
}

func (rmm *rowsMutateManager) calculateCurrentTotal() (database.CurrencyValue, error) {
	var total database.CurrencyValue

	for _, row := range rmm.rows {
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

func (rmm *rowsMutateManager) numRows() int {
	return len(rmm.rows)
}

func (rmm *rowsMutateManager) numInputs() int {
	return rmm.numRows() * rmm.numInputsPerRow()
}

func (rmm *rowsMutateManager) numInputsPerRow() int {
	return 6
}

func (rmm *rowsMutateManager) getActiveCoords() (row, col int) {
	inputsPerRow := rmm.numInputsPerRow()
	return rmm.activeInput / inputsPerRow, rmm.activeInput % inputsPerRow
}

func (rmm *rowsMutateManager) focus(direction meta.Sequence) {
	rmm.isActive = true
	numInputs := rmm.numInputs()

	switch direction {
	case meta.PREVIOUS:
		rmm.activeInput = numInputs - 1
		rmm.rows[rmm.numRows()-1].creditInput.Focus()

	case meta.NEXT:
		rmm.activeInput = 0
		rmm.rows[0].dateInput.Focus()
	}
}

// Ignores a move that would make the active input go "out of bounds"
func (rmm *rowsMutateManager) setActiveCoords(newRow, newCol int) {
	numRow := rmm.numRows()
	numPerRow := rmm.numInputsPerRow()

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
	// Have to do all this instead of leaving them all focussed, because then the cursor renders permanently
	oldRow, oldCol := rmm.getActiveCoords()
	switch oldCol {
	case 0:
		rmm.rows[oldRow].dateInput.Blur()
	case 3:
		rmm.rows[oldRow].descriptionInput.Blur()
	case 4:
		rmm.rows[oldRow].debitInput.Blur()
	case 5:
		rmm.rows[oldRow].creditInput.Blur()
	}

	rmm.activeInput = newRow*numPerRow + newCol

	switch newCol {
	case 0:
		rmm.rows[newRow].dateInput.Focus()
	case 3:
		rmm.rows[newRow].descriptionInput.Focus()
	case 4:
		rmm.rows[newRow].debitInput.Focus()
	case 5:
		rmm.rows[newRow].creditInput.Focus()
	}
}

// Converts a slice of EntryRow to a slice of EntryRowCreateView
func decompileRows(rows []database.EntryRow) ([]*rowCreator, error) {
	result := make([]*rowCreator, len(rows))

	availableLedgers := database.AvailableLedgers()
	availableAccounts := database.AvailableAccounts()

	for i, row := range rows {
		availableLedgerIndex := slices.IndexFunc(availableLedgers, func(ledger database.Ledger) bool {
			return ledger.Id == row.Ledger
		})
		if availableLedgerIndex == -1 {
			panic(fmt.Sprintf("Ledger not found for %#v", row))
		}

		ledger := availableLedgers[availableLedgerIndex]

		var account *database.Account
		if row.Account != nil {
			availableAccountIndex := slices.IndexFunc(availableAccounts, func(account database.Account) bool {
				return account.Id == *row.Account
			})
			if availableAccountIndex == -1 {
				panic(fmt.Sprintf("Account not found for %#v", row))
			}

			account = &availableAccounts[availableAccountIndex]
		}

		formRow := newRowCreator(&row.Date)

		err := formRow.ledgerInput.SetValue(ledger)
		if err != nil {
			return nil, err
		}

		err = formRow.accountInput.SetValue(account)
		if err != nil {
			return nil, err
		}

		formRow.descriptionInput.SetValue(row.Description)

		if row.Value > 0 {
			formRow.debitInput.SetValue(row.Value.String())
		} else if row.Value < 0 {
			formRow.creditInput.SetValue((-row.Value).String())
		}

		result[i] = formRow
	}

	return result, nil
}

// Checks if input is a digit or a hyphen
func validateDateInput(msg tea.KeyMsg) bool {
	// These are (likely) control flow stuff, allow it
	if len(msg.Runes) > 1 || len(msg.Runes) == 0 {
		return true
	}

	character := msg.Runes[0]

	if unicode.IsDigit(character) {
		return true
	}

	if character == '-' {
		return true
	}

	return false
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

func (rmm *rowsMutateManager) deleteRow() (*rowsMutateManager, tea.Cmd) {
	activeRow, activeCol := rmm.getActiveCoords()

	// If trying to delete the last row in the entry
	// CBA handling weird edge cases here
	if rmm.numRows() == 1 {
		return rmm, meta.MessageCmd(errors.New("cannot delete the final entryrow"))
	}

	// If about to delete the bottom-most row
	newRow, newCol := activeRow, activeCol
	if activeRow == rmm.numRows()-1 {
		newRow -= 1

		// Switch focus first to avoid index out of bounds panic when unblurring oldRow
		rmm.setActiveCoords(newRow, newCol)

		rmm.rows = append(rmm.rows[:activeRow], rmm.rows[activeRow+1:]...)
	} else {
		// Switch focus after because otherwise the to-be-deleted row gets highlighted
		rmm.rows = append(rmm.rows[:activeRow], rmm.rows[activeRow+1:]...)

		rmm.setActiveCoords(newRow, newCol)
	}

	return rmm, nil
}

func (rmm *rowsMutateManager) addRow(after bool) (*rowsMutateManager, tea.Cmd) {
	activeRow, _ := rmm.getActiveCoords()

	var newRow *rowCreator

	// If the row that the new-row-creation was triggered from had a valid date,
	// prefill it in the new row. Otherwise, just leave new row empty
	prefillDate, parseErr := database.ToDate(rmm.rows[activeRow].dateInput.Value())
	if parseErr == nil {
		newRow = newRowCreator(&prefillDate)
	} else {
		newRow = newRowCreator(nil)
	}

	newRows := make([]*rowCreator, 0, rmm.numRows()+1)

	if after {
		// Insert after activeRow
		newRows = append(newRows, rmm.rows[:activeRow+1]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, rmm.rows[activeRow+1:]...)

		rmm.rows = newRows

		rmm.setActiveCoords(activeRow+1, 0)
	} else {
		// Insert before activeRow
		newRows = append(newRows, rmm.rows[:activeRow]...)
		newRows = append(newRows, newRow)
		newRows = append(newRows, rmm.rows[activeRow:]...)

		rmm.rows = newRows

		rmm.activeInput += rmm.numInputsPerRow()
		rmm.setActiveCoords(activeRow, 0)
	}

	return rmm, nil
}

type entryMutateView interface {
	View

	getJournalInput() *itempicker.Model
	getNotesInput() *textarea.Model
	getManager() *rowsMutateManager

	getActiveInput() *int

	getColour() lipgloss.Color

	title() string
}

func entriesMutateViewUpdate(view entryMutateView, message tea.Msg) (View, tea.Cmd) {
	activeInput := view.getActiveInput()
	journalInput := view.getJournalInput()
	notesInput := view.getNotesInput()
	entryRowsManager := view.getManager()

	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		if *activeInput == ENTRIESNOTESINPUT {
			notesInput.Blur()
		}

		if *activeInput != ENTRIESROWINPUT {
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
			if *activeInput == ENTRIESROWINPUT {
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

		if *activeInput == ENTRIESNOTESINPUT {
			notesInput.Focus()
		}

		return view, nil

	case meta.NavigateMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("hjkl navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.Update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpHorizontalMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("$/_ navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.Update(message)
		*entryRowsManager = *manager

		return view, cmd

	case meta.JumpVerticalMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("'gg'/'G' navigation only works within the entryrows"))
		}

		manager, cmd := entryRowsManager.Update(message)
		*entryRowsManager = *manager

		return view, cmd

	case tea.WindowSizeMsg:
		manager := view.getManager()

		journalHeight := 3
		notesHeight := (message.Height - journalHeight) / 4
		view.getNotesInput().SetHeight(notesHeight)

		newManager, cmd := manager.Update(tea.WindowSizeMsg{
			Width: message.Width,
			// -2 for notes border
			Height: message.Height - journalHeight - notesHeight - 2,
		})
		*manager = *newManager

		return view, cmd

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch *activeInput {
		case ENTRIESJOURNALINPUT:
			*journalInput, cmd = journalInput.Update(message)
		case ENTRIESNOTESINPUT:
			*notesInput, cmd = notesInput.Update(message)
		case ENTRIESROWINPUT:
			var manager *rowsMutateManager
			manager, cmd = entryRowsManager.Update(message)
			*entryRowsManager = *manager

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", *activeInput))
		}

		return view, cmd

	case DeleteEntryRowMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("no entry row highlighted while trying to delete one"))
		}

		manager, cmd := entryRowsManager.deleteRow()
		*entryRowsManager = *manager

		return view, cmd

	case CreateEntryRowMsg:
		if *activeInput != ENTRIESROWINPUT {
			return view, meta.MessageCmd(errors.New("no entry row highlighted while trying to create one"))
		}

		var cmds []tea.Cmd

		manager, cmd := entryRowsManager.addRow(message.After)
		*entryRowsManager = *manager
		cmds = append(cmds, cmd)

		cmds = append(cmds, meta.MessageCmd(meta.SwitchModeMsg{
			InputMode: meta.INSERTMODE,
		}))

		return view, tea.Batch(cmds...)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func entriesMutateViewView(view entryMutateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(view.getColour()).Padding(0, 1)
	result.WriteString(titleStyle.Render(view.title()))

	result.WriteString("\n\n")

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(view.getColour())

	inputs := []viewable{view.getJournalInput(), view.getNotesInput(), view.getManager()}
	names := []string{"Journal", "Notes", ""}

	styles := slices.Repeat([]lipgloss.Style{sectionStyle}, len(names))
	styles[*view.getActiveInput()] = highlightStyle

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	for i := range names {
		if names[i] == "" {
			result.WriteString(sectionStyle.Render(inputs[i].View()))
		} else {
			result.WriteString(lipgloss.JoinHorizontal(
				lipgloss.Top,
				sectionStyle.Width(maxNameColWidth).Render(names[i]),
				" ",
				styles[i].Render(inputs[i].View()),
			))
		}

		result.WriteString("\n")
	}

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}

func entriesMutateViewMotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	// Default navigation
	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	// Create/delete rows
	normalMotions.Insert(meta.Motion{"d", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "d"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"V", "D"}, DeleteEntryRowMsg{})
	normalMotions.Insert(meta.Motion{"o"}, CreateEntryRowMsg{After: true})
	normalMotions.Insert(meta.Motion{"O"}, CreateEntryRowMsg{After: false})

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

	return meta.MotionSet{
		Normal: normalMotions,
	}
}

func entriesMutateViewCommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

type entryDeleteView struct {
	DB *sqlx.DB

	modelId int // only for retrieving the model itself initially
	model   database.Entry

	rows    []*database.EntryRow
	journal *database.Journal

	colour lipgloss.Color
}

func NewEntryDeleteView(DB *sqlx.DB, modelId int) *entryDeleteView {
	return &entryDeleteView{
		DB: DB,

		modelId: modelId,

		colour: meta.ENTRIESCOLOUR,
	}
}

func (dv *entryDeleteView) Init() tea.Cmd {
	// Can't load journal yet, we only know journal ID when entry is loaded
	entryCmd := database.MakeLoadEntriesDetailCmd(dv.DB, dv.modelId)
	rowsCmd := database.MakeSelectEntryRowsCmd(dv.DB, dv.modelId)

	return tea.Batch(entryCmd, rowsCmd)
}

func (dv *entryDeleteView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYMODEL:
			dv.model = message.Data.(database.Entry)

			availableJournals := database.AvailableJournals()
			journalIndex := slices.IndexFunc(availableJournals, func(j database.Journal) bool {
				return j.Id == dv.model.Journal
			})
			if journalIndex == -1 {
				return dv, meta.MessageCmd(fmt.Errorf("couldn't find journal %d in cache", dv.model.Journal))
			}

			// Don't reference original journal directly to ensure cache is never mutated
			journal := availableJournals[journalIndex]
			dv.journal = &journal

		case meta.ENTRYROWMODEL:
			rows := message.Data.([]database.EntryRow)

			dv.rows = make([]*database.EntryRow, len(rows))
			for i, row := range rows {
				dv.rows[i] = &row
			}
		}

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteEntry(dv.DB, dv.model.Id)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted entry \"%d\"", dv.modelId,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		// TODO

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *entryDeleteView) View() string {
	return genericDeleteViewView(dv)
}

func (dv *entryDeleteView) Type() meta.ViewType {
	return meta.DELETEVIEWTYPE
}

func (dv *entryDeleteView) AllowsInsertMode() bool {
	return false
}

func (dv *entryDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL:    {},
		meta.ENTRYROWMODEL: {},
	}
}

func (dv *entryDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *entryDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *entryDeleteView) Reload() View {
	return NewEntryDeleteView(dv.DB, dv.modelId)
}

func (dv *entryDeleteView) title() string {
	return fmt.Sprintf("Delete entry %s", dv.model.String())
}

func (dv *entryDeleteView) inputValues() []string {
	var result []string

	if dv.journal == nil {
		result = append(result, strconv.Itoa(dv.model.Journal))
	} else {
		result = append(result, dv.journal.Name)
	}

	result = append(result, dv.model.Notes.Collapse())

	result = append(result, fmt.Sprintf("%d", len(dv.rows)))

	result = append(result, database.CalculateSize(dv.rows).String())

	return result
}

func (dv *entryDeleteView) inputNames() []string {
	return []string{"Journal", "Notes", "# rows", "Entry size"}
}

func (dv *entryDeleteView) getColour() lipgloss.Color {
	return dv.colour
}

func (dv *entryDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
