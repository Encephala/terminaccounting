package view

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"terminaccounting/bubbles/list"
	"terminaccounting/database"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type genericDetailView interface {
	View

	title() string
	metadata() metadata
	getWidth() int

	getDB() *sqlx.DB

	getCanReconcile() bool

	getViewer() *entryRowViewer
}

func genericDetailViewUpdate(gdv genericDetailView, message tea.Msg) (View, tea.Cmd) {
	viewer := gdv.getViewer()

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		newViewer, cmd := viewer.Update(tea.WindowSizeMsg{
			Width: message.Width,
			// -1 for the title, reconcilableness info
			Height: message.Height - 1,
		})
		*viewer = *newViewer

		return gdv, cmd

	case meta.DataLoadedMsg, meta.NavigateMsg, *meta.JumpVerticalMsg, *meta.UpdateSearchMsg:
		newViewer, cmd := viewer.Update(message)
		*viewer = *newViewer

		return gdv, cmd

	case meta.ToggleShowReconciledMsg:
		oldItems := viewer.list.Items()
		oldShownRows := make([]*database.EntryRow, len(oldItems))
		oldActiveIndex := viewer.list.ActiveIndex()
		for i, item := range oldItems {
			oldShownRows[i] = item.(*database.EntryRow)
		}

		viewer.showReconciled = !viewer.showReconciled
		viewer.updateViewRows()

		if len(viewer.list.Items()) == 0 || len(oldShownRows) == 0 {
			return gdv, nil
		}

		// If there was previously an active row, set a new row active that is reasonably close to the old one
		// Find (or try to) the closest row equal to or before the previous active row that is still being shown
		for i := oldActiveIndex; i >= 0; i-- {
			found := viewer.setActiveRow(oldShownRows[i])

			if found {
				return gdv, nil
			}
		}

		// If no closest row before found (i.e. active row was one of the first rows and is now hidden)
		for i := oldActiveIndex + 1; i < len(oldShownRows); i++ {
			found := viewer.setActiveRow(oldShownRows[i])

			if found {
				return gdv, nil
			}
		}

		panic("this never happens due to the above check that the new items are non-empty")

	case meta.ReconcileMsg:
		if !gdv.getCanReconcile() {
			return gdv, meta.MessageCmd(errors.New("reconciling is disabled in this view"))
		}

		activeEntryRow := (*viewer.list.ActiveItem()).(*database.EntryRow)

		if activeEntryRow == nil {
			return gdv, meta.MessageCmd(errors.New("there are no rows to reconcile"))
		}

		activeEntryRow.Reconciled = !activeEntryRow.Reconciled
		// If the last row was just reconciled and it is now hidden, set activeRow to the new last row
		activeIndex := viewer.list.ActiveIndex()

		if !viewer.showReconciled && activeIndex == len(viewer.list.Items())-1 {
			activeIndex = max(0, activeIndex-1)
		}

		viewer.updateViewRows()

		return gdv, nil

	case meta.CommitMsg:
		if !viewer.rowsAreChanged() {
			return gdv, meta.MessageCmd(meta.NotificationMessageMsg{Message: "there are no changes in reconciliation to commit"})
		}

		total := database.CalculateTotal(viewer.getReconciledRows())
		if total != 0 {
			return gdv, meta.MessageCmd(fmt.Errorf("total of reconciled rows not 0 but %s", total))
		}

		changed, err := database.SetReconciled(gdv.getDB(), viewer.rows)
		if err != nil {
			return gdv, meta.MessageCmd(err)
		}

		notification := meta.NotificationMessageMsg{Message: fmt.Sprintf("set reconciled status, updated %d rows", changed)}

		// Reset dv.originalRows
		for i, row := range viewer.rows {
			viewer.originalRows[i] = *row
		}

		return gdv, meta.MessageCmd(notification)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func genericDetailViewView(gdv genericDetailView) string {
	var result strings.Builder

	result.WriteString(renderHeader(gdv.title(), gdv.metadata(), gdv.getWidth()))
	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("Showing reconciled rows: %s", meta.RenderBoolean(gdv.getViewer().showReconciled)))
	result.WriteString("\n")

	result.WriteString(fmt.Sprintf("Reconciling enabled: %s", meta.RenderBoolean(gdv.getCanReconcile())))
	result.WriteString("\n\n")

	result.WriteString(gdv.getViewer().View())

	return result.String()
}

func genericDetailViewMotionSet() meta.Trie[tea.Msg] {
	var motions meta.Trie[tea.Msg]

	motions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	motions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	motions.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{Down: false})
	motions.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{Down: true})

	motions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	motions.Insert(meta.Motion{"s", "r"}, meta.ToggleShowReconciledMsg{}) // [S]how [R]econciled
	motions.Insert(meta.Motion{"enter"}, meta.ReconcileMsg{})

	return motions
}

func genericDetailViewCommandSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return result
}

func makeGoToEntryDetailViewCmd(DB *sqlx.DB, activeEntryRow *database.EntryRow) tea.Cmd {
	return func() tea.Msg {
		if activeEntryRow == nil {
			return meta.MessageCmd(errors.New("there is no row to view details of"))
		}

		entryId := activeEntryRow.Entry

		// Do the database query for the entry here, because it is a command and thus asynchronous
		entry, err := database.SelectEntry(DB, entryId)

		if err != nil {
			return meta.MessageCmd(err)
		}

		// Stupid go not allowing to reference a const
		targetApp := meta.ENTRIESAPP
		return meta.SwitchAppViewMsg{App: &targetApp, ViewType: meta.DETAILVIEWTYPE, Data: entry}
	}
}

type entryRowViewer struct {
	width, height   int
	highlightColour lipgloss.Color

	list list.Model

	// All the rows in this viewer
	rows []*database.EntryRow
	// The original rows in the viewer (for comparing reconciled-state)
	originalRows []database.EntryRow

	showReconciled bool

	headers   []string
	colWidths []int
}

func newEntryRowViewer(colour lipgloss.Color) *entryRowViewer {
	result := &entryRowViewer{
		highlightColour: colour,

		list: list.New(0, 0),

		headers: []string{"Date", "Ledger", "Account", "Description", "Debit", "Credit", "Reconciled"},
	}

	return result
}

func (erv *entryRowViewer) Update(message tea.Msg) (*entryRowViewer, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		erv.width = message.Width
		erv.height = message.Height

		erv.list.Update(tea.WindowSizeMsg{
			Width: message.Width,
			// -8 for the total rows and their vertical margin
			Height: message.Height - 8,
		})

		erv.calculateColumnWidths()

		return erv, nil

	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYROWMODEL:
			data := message.Data.([]database.EntryRow)

			erv.originalRows = make([]database.EntryRow, len(data))
			erv.rows = make([]*database.EntryRow, len(data))

			for i, row := range data {
				erv.originalRows[i] = row
				erv.rows[i] = &row
			}

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

		erv.updateViewRows()
		erv.calculateColumnWidths()

		return erv, nil

	case meta.NavigateMsg:
		erv.list.Navigate(message.Direction == meta.DOWN)

		return erv, nil

	case meta.JumpVerticalMsg:
		erv.list.Jump(message.Down)

		return erv, nil

	case meta.UpdateSearchMsg:
		erv.list.SetFilter(message.Query)

		return erv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (erv *entryRowViewer) View() string {
	var result strings.Builder

	result.WriteString(erv.list.View())

	result.WriteString(fmt.Sprintf("Total: %s", database.CalculateTotal(erv.rows)))

	result.WriteString("\n")

	if erv.rowsAreChanged() {
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

func (erv *entryRowViewer) calculateColumnWidths() {
	dateWidth := 10 // This is simply the width of a date field
	reconciledWidth := len("Reconciled")

	var colWidths []int
	remainingWidth := erv.width - dateWidth - reconciledWidth
	descriptionWidth := remainingWidth / 3

	// -12 because of the 2-wide padding between columns, 6x
	// /4 because there are four other columns
	othersWidth := (remainingWidth - descriptionWidth - 12) / 4
	colWidths = []int{dateWidth, othersWidth, othersWidth, descriptionWidth, othersWidth, othersWidth, reconciledWidth}

	erv.colWidths = colWidths
}

// Takes the rows, and depending on showReconciled state, updates the list's items
func (erv *entryRowViewer) updateViewRows() {
	var viewRows []list.Item
	if erv.showReconciled {
		for _, row := range erv.rows {
			viewRows = append(viewRows, row)
		}
	} else {
		for _, row := range erv.getUnreconciledRows() {
			viewRows = append(viewRows, row)
		}
	}

	erv.list.SetItems(viewRows)
}

func (erv *entryRowViewer) setActiveRow(entryRow *database.EntryRow) (found bool) {
	index := slices.IndexFunc(erv.list.Items(), func(row list.Item) bool { return row == entryRow })

	if index == -1 {
		// This means that the row that was highlighted is reconciled and is now hidden, all good
		return false
	}

	erv.list.SetActiveIndex(index)

	return true
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
