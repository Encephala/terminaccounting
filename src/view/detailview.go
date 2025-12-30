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

type genericDetailView interface {
	View

	title() string

	getViewer() *entryRowViewer

	getColours() meta.AppColours
}

func genericDetailViewUpdate(gdv genericDetailView, message tea.Msg) (View, tea.Cmd) {
	viewer := gdv.getViewer()

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		newViewer, cmd := viewer.Update(tea.WindowSizeMsg{
			// -4 for padding on each side
			Width: message.Width - 4,
			// -4 for the title and table header (header is not considered for table width)
			// -6 to for the total rows their vertical margin
			Height: message.Height - 4 - 6,
		})
		*viewer = *newViewer

		viewer.calculateColumnWidths()

		return gdv, cmd

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYROWMODEL:
			data := message.Data.([]database.EntryRow)

			viewer.originalRows = make([]database.EntryRow, len(data))
			viewer.rows = make([]*database.EntryRow, len(data))

			for i, row := range data {
				viewer.originalRows[i] = row
				viewer.rows[i] = &row
			}

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

		viewer.updateViewRows()

		return gdv, nil

	case meta.NavigateMsg:
		newViewer, cmd := viewer.Update(message)
		*viewer = *newViewer

		return gdv, cmd

	case meta.ToggleShowReconciledMsg:
		if !viewer.canReconcile {
			return gdv, meta.MessageCmd(errors.New("reconciling is disabled, no reconciled rows to show"))
		}

		viewer.showReconciled = !viewer.showReconciled

		viewer.updateViewRows()

		if activeEntryRow := viewer.activeEntryRow(); activeEntryRow != nil {
			viewer.activateEntryRow(activeEntryRow)
		}

		return gdv, nil

	case meta.ReconcileMsg:
		if !viewer.canReconcile {
			return gdv, meta.MessageCmd(errors.New("reconciling is disabled"))
		}

		activeEntryRow := viewer.activeEntryRow()

		if activeEntryRow == nil {
			return gdv, meta.MessageCmd(errors.New("there are no rows to reconcile"))
		}

		activeEntryRow.Reconciled = !activeEntryRow.Reconciled
		if !viewer.showReconciled && viewer.activeRow == len(viewer.viewRows)-1 {
			viewer.activeRow = max(0, viewer.activeRow-1)
		}

		viewer.updateViewRows()

		viewer.showReconciledTotal = viewer.rowsAreChanged()

		return gdv, nil

	case meta.CommitMsg:
		if !viewer.rowsAreChanged() {
			return gdv, meta.MessageCmd(meta.NotificationMessageMsg{Message: "there are no changes in reconciliation to commit"})
		}

		total := database.CalculateTotal(viewer.getReconciledRows())
		if total != 0 {
			return gdv, meta.MessageCmd(fmt.Errorf("total of reconciled rows not 0 but %s", total))
		}

		changed, err := database.SetReconciled(viewer.rows)
		if err != nil {
			return gdv, meta.MessageCmd(err)
		}

		notification := meta.NotificationMessageMsg{Message: fmt.Sprintf("set reconciled status, updated %d rows", changed)}

		// Reset dv.originalRows
		for i, row := range viewer.rows {
			viewer.originalRows[i] = *row
			viewer.showReconciledTotal = false
		}

		return gdv, meta.MessageCmd(notification)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func genericDetailViewView(gdv genericDetailView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(gdv.getColours().Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(gdv.title()))

	result.WriteString("\n")

	if gdv.getViewer().canReconcile {
		result.WriteString(fmt.Sprintf("Showing reconciled rows: %s", renderBoolean(gdv.getViewer().showReconciled)))
	} else {
		result.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#777777")).Render("Reconciling disabled"))
	}

	result.WriteString("\n\n")

	result.WriteString(gdv.getViewer().View())

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}

func genericDetailViewMotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"s", "r"}, meta.ToggleShowReconciledMsg{}) // [S]how [R]econciled
	normalMotions.Insert(meta.Motion{"enter"}, meta.ReconcileMsg{})

	return meta.MotionSet{Normal: normalMotions}
}

func genericDetailViewCommandSet() meta.CommandSet {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(result)
}

func makeGoToEntryDetailViewCmd(activeEntryRow *database.EntryRow) tea.Cmd {
	return func() tea.Msg {
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

	rows         []*database.EntryRow
	originalRows []database.EntryRow

	activeRow      int
	highlightStyle lipgloss.Style

	canReconcile        bool
	showReconciled      bool
	showReconciledTotal bool

	headers   []string
	colWidths []int
}

func newEntryRowViewer(colours meta.AppColours) *entryRowViewer {
	result := &entryRowViewer{
		viewport: viewport.New(0, 0),

		highlightStyle: lipgloss.NewStyle().Foreground(colours.Foreground),

		canReconcile: false,

		colWidths: []int{0, 0, 0, 0, 0, 0},
		headers:   []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit"},
	}

	return result
}

func (erv *entryRowViewer) Update(message tea.Msg) (*entryRowViewer, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
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

	result.WriteString("\n\n")

	result.WriteString(fmt.Sprintf("Total: %s", database.CalculateTotal(erv.rows)))

	result.WriteString("\n")

	if erv.canReconcile && erv.showReconciledTotal {
		totalReconciled := database.CalculateTotal(erv.getReconciledRows())

		var totalReconciledRendered string
		if totalReconciled == 0 {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
			totalReconciledRendered = style.Render(fmt.Sprintf("%s", totalReconciled))
		} else {
			totalReconciledRendered = fmt.Sprintf("%s", totalReconciled)
		}

		result.WriteString(fmt.Sprintf("Reconciled total: %s", totalReconciledRendered))

		result.WriteString("\n")
	}

	result.WriteString(fmt.Sprintf("Size: %s", database.CalculateSize(erv.rows)))

	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("# rows: %d", len(erv.rows)))

	return result.String()
}

func (erv *entryRowViewer) setCanReconcile(canReconcile bool) {
	erv.canReconcile = canReconcile

	if erv.canReconcile {
		erv.headers = []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit", "Reconciled"}
	} else {
		erv.headers = []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit"}
	}

	erv.updateViewRows()
	erv.calculateColumnWidths()
}

func (erv *entryRowViewer) calculateColumnWidths() {
	dateWidth := 10 // This is simply the width of a date field
	reconciledWidth := len("Reconciled")

	var colWidths []int
	if erv.canReconcile {
		// -4 because of left/right margin
		remainingWidth := erv.width - dateWidth - reconciledWidth - 4
		descriptionWidth := remainingWidth / 3

		// -12 because of the 2-wide padding between columns, 6x
		// /4 because there are four other columns
		othersWidth := (remainingWidth - descriptionWidth - 12) / 4
		colWidths = []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth, reconciledWidth}
	} else {
		// -4 because of left/right margin
		remainingWidth := erv.width - dateWidth - 4
		descriptionWidth := remainingWidth / 3

		// -10 because of the 2-wide padding between columns, 5x
		// /4 because there are four other columns
		othersWidth := (remainingWidth - descriptionWidth - 10) / 4
		colWidths = []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth}
	}

	erv.colWidths = colWidths
}

func (erv *entryRowViewer) updateViewRows() {
	var shownRows []*database.EntryRow
	if erv.showReconciled {
		shownRows = erv.rows
	} else {
		shownRows = erv.getUnreconciledRows()
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

		if erv.canReconcile {
			viewRow = append(viewRow, lipgloss.NewStyle().AlignHorizontal(lipgloss.Center).Render(renderBoolean(row.Reconciled)))
		}

		viewRows = append(viewRows, viewRow)
	}

	erv.viewRows = viewRows
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
	if len(erv.viewRows) == 0 {
		return nil
	}

	return erv.rows[erv.activeRow]
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
	index := slices.IndexFunc(erv.rows, func(row *database.EntryRow) bool { return row == entryRow })

	if index == -1 {
		// This means that the row that was highlighted is reconciled and is now hidden, all good
		return
	}

	erv.activeRow = index
	erv.scrollViewport()
}

func (erv *entryRowViewer) getReconciledRows() []*database.EntryRow {
	var result []*database.EntryRow

	for _, row := range erv.rows {
		if row.Reconciled {
			result = append(result, row)
		}
	}

	return result
}

func (erv *entryRowViewer) getUnreconciledRows() []*database.EntryRow {
	var result []*database.EntryRow

	for _, row := range erv.rows {
		if !row.Reconciled {
			result = append(result, row)
		}
	}

	return result
}

func (erv *entryRowViewer) rowsAreChanged() bool {
	return !slices.EqualFunc(
		erv.rows,
		erv.originalRows,
		func(row *database.EntryRow, originalRow database.EntryRow) bool { return *row == originalRow },
	)
}
