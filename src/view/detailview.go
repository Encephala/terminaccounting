package view

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// A generic, placeholder view that just renders all entries on a ledger/account in a list.
type DetailView struct {
	app meta.App

	// The ledger/account etc. whose rows are being shown
	modelId   int
	modelName string

	originalRows   []database.EntryRow
	rows           []*database.EntryRow
	viewer         *entryRowViewer
	showReconciled bool

	showTotalReconciled bool
}

func NewDetailView(app meta.App, modelId int, modelName string) *DetailView {
	return &DetailView{
		app: app,

		modelId:   modelId,
		modelName: modelName,

		viewer:         newEntryRowViewer(app.Colours()),
		showReconciled: false,
	}
}

func (dv *DetailView) Init() tea.Cmd {
	// TODO: Also show the model metadata and not just the rows?
	var cmds []tea.Cmd
	cmds = append(cmds, dv.app.MakeLoadRowsCmd(dv.modelId))

	cmds = append(cmds, database.MakeSelectLedgersCmd(dv.app.Type()))
	cmds = append(cmds, database.MakeSelectAccountsCmd(dv.app.Type()))

	cmds = append(cmds, dv.viewer.Init())

	return tea.Batch(cmds...)
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		var cmd tea.Cmd

		dv.viewer, cmd = dv.viewer.Update(tea.WindowSizeMsg{
			// -4 for padding on each side
			Width: message.Width - 4,
			// -4 for the title and table header (header is not considered for table width)
			// -4 to for the total rows their vertical margin
			Height: message.Height - 4 - 4,
		})

		return dv, cmd

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYROWMODEL:
			for _, row := range message.Data.([]database.EntryRow) {
				dv.originalRows = append(dv.originalRows, row)
				dv.rows = append(dv.rows, &row)
			}

		case meta.ACCOUNTMODEL:
			database.AvailableAccounts = message.Data.([]database.Account)

		case meta.LEDGERMODEL:
			database.AvailableLedgers = message.Data.([]database.Ledger)

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

		dv.updateViewRows()

		return dv, nil

	case meta.NavigateMsg:
		var cmd tea.Cmd
		dv.viewer, cmd = dv.viewer.Update(message)

		return dv, cmd

	case meta.ToggleShowReconciledMsg:
		activeEntryRow := dv.viewer.activeEntryRow()

		dv.showReconciled = !dv.showReconciled

		dv.updateViewRows()

		if activeEntryRow != nil {
			dv.viewer.activateEntryRow(activeEntryRow)
		}

		return dv, nil

	case meta.ReconcileMsg:
		activeEntryRow := dv.viewer.activeEntryRow()

		if activeEntryRow == nil {
			return dv, meta.MessageCmd(errors.New("there are no rows to reconcile"))
		}

		activeEntryRow.Reconciled = !activeEntryRow.Reconciled
		// TODO: this manual inspection of viewer from the outside feels wrong, improper separation of concerns
		if !dv.showReconciled && dv.viewer.activeRow == len(dv.viewer.viewRows)-1 {
			dv.viewer.activeRow = max(0, dv.viewer.activeRow-1)
		}
		dv.updateViewRows()

		if dv.rowsAreChanged() {
			dv.showTotalReconciled = true
		} else {
			dv.showTotalReconciled = false
		}

		return dv, nil

	case meta.CommitMsg:
		if !dv.rowsAreChanged() {
			return dv, meta.MessageCmd(meta.NotificationMessageMsg{Message: "there are no changes in reconciliation to commit"})
		}

		total := database.CalculateTotal(dv.getReconciledRows())
		if total != 0 {
			return dv, meta.MessageCmd(fmt.Errorf("total of reconciled rows not 0 but %s", total))
		}

		changed, err := database.SetReconciled(dv.rows)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		notification := meta.NotificationMessageMsg{Message: fmt.Sprintf("set reconciled status, updated %d rows", changed)}

		// Reset dv.originalRows
		for i, row := range dv.rows {
			dv.originalRows[i] = *row
			dv.showTotalReconciled = false
		}

		return dv, meta.MessageCmd(notification)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *DetailView) View() string {
	var result strings.Builder

	marginLeftStyle := lipgloss.NewStyle().MarginLeft(2)

	titleStyle := marginLeftStyle.Background(dv.app.Colours().Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(fmt.Sprintf("%s Details: %s", dv.app.Name(), dv.modelName)))

	result.WriteString("\n")

	result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Showing reconciled rows: %s", renderBoolean(dv.showReconciled))))

	result.WriteString("\n\n")

	result.WriteString(marginLeftStyle.Render(dv.viewer.View()))

	result.WriteString("\n\n")

	result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Total: %s", database.CalculateTotal(dv.rows))))

	result.WriteString("\n")

	if dv.showTotalReconciled {
		totalReconciled := database.CalculateTotal(dv.getReconciledRows())
		var totalReconciledRendered string
		if totalReconciled == 0 {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
			totalReconciledRendered = style.Render(fmt.Sprintf("%s", totalReconciled))
		} else {
			totalReconciledRendered = fmt.Sprintf("%s", totalReconciled)
		}
		result.WriteString(marginLeftStyle.Render(fmt.Sprintf("Reconciled total: %s", totalReconciledRendered)))
	}

	return result.String()
}

func (dv *DetailView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYROWMODEL: {},
		meta.ACCOUNTMODEL:  {},
		meta.LEDGERMODEL:   {},
	}
}

func (dv *DetailView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "x"}, meta.SwitchViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})

	normalMotions.Insert(meta.Motion{"g", "e"}, meta.SwitchViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	normalMotions.Insert(meta.Motion{"s", "r"}, meta.ToggleShowReconciledMsg{}) // [S]how [R]econciled
	normalMotions.Insert(meta.Motion{"enter"}, meta.ReconcileMsg{})

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *DetailView) CommandSet() meta.CommandSet {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(result)
}

func (dv *DetailView) Reload() View {
	return NewDetailView(dv.app, dv.modelId, dv.modelName)
}

func (dv *DetailView) rowsAreChanged() bool {
	return !slices.EqualFunc(
		dv.rows,
		dv.originalRows,
		func(row *database.EntryRow, originalRow database.EntryRow) bool { return *row == originalRow },
	)
}

func (dv *DetailView) updateViewRows() {
	var shownRows []*database.EntryRow
	if dv.showReconciled {
		shownRows = dv.rows
	} else {
		shownRows = dv.getUnreconciledRows()
	}

	var viewRows [][]string
	for _, row := range shownRows {
		var viewRow []string
		viewRow = append(viewRow, row.Date.String())

		var ledger, account string

		availableLedgerIndex := slices.IndexFunc(database.AvailableLedgers, func(ledger database.Ledger) bool {
			return ledger.Id == row.Ledger
		})
		if availableLedgerIndex != -1 {
			ledger = database.AvailableLedgers[availableLedgerIndex].String()
		} else {
			ledger = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Italic(true).Render("error")
		}

		if row.Account == nil {
			account = lipgloss.NewStyle().Italic(true).Render("None")
		} else {
			availableAccountIndex := slices.IndexFunc(database.AvailableAccounts, func(account database.Account) bool {
				return account.Id == *row.Account
			})

			if availableAccountIndex != -1 {
				account = database.AvailableAccounts[availableAccountIndex].String()
			} else {
				ledger = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Italic(true).Render("error")
			}
		}

		viewRow = append(viewRow, ledger)
		viewRow = append(viewRow, account)
		viewRow = append(viewRow, row.Description)
		if row.Value > 0 {
			viewRow = append(viewRow, row.Value.String())
			viewRow = append(viewRow, "")
		} else {
			viewRow = append(viewRow, "")
			viewRow = append(viewRow, (-row.Value).String())
		}

		viewRow = append(viewRow, "    "+renderBoolean(row.Reconciled))

		viewRows = append(viewRows, viewRow)
	}

	dv.viewer.setRows(viewRows, shownRows)
}

func (dv *DetailView) getUnreconciledRows() []*database.EntryRow {
	var result []*database.EntryRow

	for _, row := range dv.rows {
		if !row.Reconciled {
			result = append(result, row)
		}
	}

	return result
}

func (dv *DetailView) getReconciledRows() []*database.EntryRow {
	var result []*database.EntryRow

	for _, row := range dv.rows {
		if row.Reconciled {
			result = append(result, row)
		}
	}

	return result
}

func (dv *DetailView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		activeEntryRow := dv.viewer.activeEntryRow()

		if activeEntryRow == nil {
			return meta.MessageCmd(errors.New("there is no row to view details of"))
		}

		entryId := activeEntryRow.Entry

		// Do the database query for the entry here, because it is a command and thus asynchronous
		entry, err := database.SelectEntry(entryId)

		if err != nil {
			return meta.MessageCmd(err)
		}

		// Stupid go not allowing to reference a const
		targetApp := meta.ENTRIESAPP
		return meta.SwitchViewMsg{App: &targetApp, ViewType: meta.DETAILVIEWTYPE, Data: entry}
	}
}

type entryRowViewer struct {
	width, height int

	viewport viewport.Model
	viewRows [][]string
	// The EntryRows that are being shown
	entryRows []*database.EntryRow

	activeRow      int
	highlightStyle lipgloss.Style

	headers   []string
	colWidths []int
}

func newEntryRowViewer(colours meta.AppColours) *entryRowViewer {
	return &entryRowViewer{
		viewport: viewport.New(0, 0),

		highlightStyle: lipgloss.NewStyle().Foreground(colours.Foreground),

		headers: []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit", "Reconciled"},
	}
}

func (erv *entryRowViewer) Init() tea.Cmd {
	return nil
}

func (erv *entryRowViewer) Update(message tea.Msg) (*entryRowViewer, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		erv.colWidths = erv.calculateColumnWidths(message.Width)

		erv.width = message.Width
		erv.height = message.Height

		erv.viewport.Width = message.Width
		erv.viewport.Height = message.Height

		return erv, nil

	case meta.NavigateMsg:
		switch message.Direction {
		case meta.DOWN:
			if erv.activeRow != erv.viewport.TotalLineCount()-1 {
				erv.activeRow++
			}

		case meta.UP:
			if erv.activeRow != 0 {
				erv.activeRow--
			}

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

		erv.scrollViewport()

		return erv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (erv *entryRowViewer) View() string {
	var result strings.Builder

	erv.setViewportContent()

	result.WriteString(erv.renderRow(erv.headers, false))

	result.WriteString("\n")

	result.WriteString(erv.viewport.View())

	return result.String()
}

func (erv *entryRowViewer) scrollViewport() {
	if erv.activeRow >= erv.viewport.YOffset+erv.height {
		erv.viewport.ScrollDown(erv.activeRow - erv.viewport.YOffset - erv.height + 1)
	}

	if erv.activeRow < erv.viewport.YOffset {
		erv.viewport.ScrollUp(erv.viewport.YOffset - erv.activeRow)
	}
}

func (erv *entryRowViewer) activeEntryRow() *database.EntryRow {
	if len(erv.entryRows) == 0 {
		return nil
	}

	return erv.entryRows[erv.activeRow]
}

func (erv *entryRowViewer) calculateColumnWidths(totalWidth int) []int {
	dateWidth := 10 // This is simply the width of a date field
	reconciledWidth := len("Reconciled")

	// -2 because of left/right padding
	remainingWidth := totalWidth - dateWidth - reconciledWidth - 4
	descriptionWidth := remainingWidth / 3

	// -12 because of the 2-wide padding between columns, 6x
	othersWidth := (remainingWidth - descriptionWidth - 12) / 4
	colWidths := []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth, reconciledWidth}

	return colWidths
}

func (erv *entryRowViewer) setRows(viewRows [][]string, rawRows []*database.EntryRow) {
	erv.viewRows = viewRows
	erv.entryRows = rawRows
}

func (erv *entryRowViewer) setViewportContent() {
	var content []string

	for i, row := range erv.viewRows {
		doHighlight := i == erv.activeRow
		content = append(content, erv.renderRow(row, doHighlight))
	}

	erv.viewport.SetContent(strings.Join(content, "\n"))
}

func (erv *entryRowViewer) renderRow(values []string, highlight bool) string {
	if len(erv.colWidths) != len(values) {
		panic("You absolute dingus")
	}

	var result strings.Builder

	for i := range values {
		style := lipgloss.NewStyle()
		if highlight {
			style = erv.highlightStyle
		}

		style = style.Width(erv.colWidths[i])

		if i != len(values)-1 {
			style = style.MarginRight(2)
		}

		result.WriteString(style.Render(ansi.Truncate(values[i], erv.colWidths[i], "…")))
	}

	return result.String()
}

func (erv *entryRowViewer) activateEntryRow(entryRow *database.EntryRow) {
	index := slices.IndexFunc(erv.entryRows, func(row *database.EntryRow) bool { return row == entryRow })

	if index == -1 {
		// This means that the row that was highlighted is reconciled and is now hidden, all good
		return
	}

	erv.activeRow = index
	erv.scrollViewport()
}
