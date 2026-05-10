package modals

import (
	"fmt"
	"slices"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type searchItem struct {
	model any
}

func (item *searchItem) Render(isActive bool) string {
	var color lipgloss.Color
	var text string

	switch model := item.model.(type) {
	case database.Ledger:
		color = meta.LEDGERSCOLOUR
		text = model.String()

	case database.Account:
		color = meta.ACCOUNTSCOLOUR
		text = model.String()

	case database.Journal:
		color = meta.JOURNALSCOLOUR
		text = model.String()

	case database.Entry:
		color = meta.ENTRIESCOLOUR
		text = "Entry " + model.String()

	case database.EntryRow:
		color = meta.ENTRIESCOLOUR
		// TODO
		text = "entryRow"

	default:
		panic(fmt.Sprintf("unexpected type: %#v", model))
	}

	if isActive {
		color = lipgloss.Color("7")
	}

	return lipgloss.NewStyle().Foreground(color).Render(text)
}

type globalSearchModal struct {
	DB *sqlx.DB

	searchQuery string

	viewport  viewport.Model
	activeRow int

	items []searchItem
}

func newGlobalSearchModal(DB *sqlx.DB) *globalSearchModal {
	return &globalSearchModal{
		DB: DB,

		viewport: viewport.New(0, 0),
	}
}

func (sm *globalSearchModal) Init() tea.Cmd {
	return makeSelectAllDataCmd(sm.DB)
}

func (sm *globalSearchModal) Update(message tea.Msg) (Modal, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		sm.viewport.Width = message.Width
		sm.viewport.Height = message.Height - 2

		return sm, nil

	case meta.NavigateMsg:
		if message.Direction == meta.DOWN {
			sm.activeRow = min(sm.activeRow+1, len(sm.items)-1)
		} else {
			sm.activeRow = max(sm.activeRow-1, 0)
		}

		return sm, nil

	case meta.UpdateSearchMsg:
		var cmd tea.Cmd
		sm.searchQuery = message.Query

		// TODO: filter items
		// Also make sure activeRow stays in sane state

		return sm, cmd

	case meta.GlobalSearchDataLoadedMsg:
		sm.items = make([]searchItem, len(message.Data))

		for i, item := range message.Data {
			sm.items[i] = item.(searchItem)
		}

		return sm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (sm *globalSearchModal) View() string {
	var result strings.Builder

	result.WriteString("Query: " + sm.searchQuery)
	result.WriteString("\n\n")

	sm.updateViewportContent()

	result.WriteString(sm.viewport.View())

	return result.String()
}

func (sm *globalSearchModal) updateViewportContent() {
	var lines []string

	for i, item := range sm.items {
		lines = append(lines, item.Render(sm.activeRow == i))
	}

	sm.viewport.SetContent(strings.Join(lines, "\n"))
}

func (sm *globalSearchModal) AllowsInsertMode() bool {
	return false
}

func (sm *globalSearchModal) AllowsSearchMode() bool {
	return true
}

func (sm *globalSearchModal) MotionSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	result.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	// Closures to the rescue once more
	var gotoDetailViewCmd tea.Cmd
	gotoDetailViewCmd = func() tea.Msg {
		var appType *meta.AppType
		var data any

		activeItem := sm.items[sm.activeRow]
		switch model := activeItem.model.(type) {
		case database.Ledger:
			tmp := meta.LEDGERSAPP
			appType = &tmp
			data = model

		case database.Account:
			tmp := meta.ACCOUNTSAPP
			appType = &tmp
			data = model

		case database.Journal:
			tmp := meta.JOURNALSAPP
			appType = &tmp
			data = model

		case database.Entry:
			tmp := meta.ENTRIESAPP
			appType = &tmp
			data = model

		case database.EntryRow:
			tmp := meta.ENTRIESAPP
			appType = &tmp

			entry, err := database.SelectEntry(sm.DB, model.Entry)
			if err != nil {
				return fmt.Errorf("Failed to go to entry detail view: %s", err)
			}

			data = entry

		default:
			panic(fmt.Sprintf("unexpected type: %#v", model))
		}

		return meta.SwitchAppViewMsg{
			App:      appType,
			ViewType: meta.DETAILVIEWTYPE,
			Data:     data,
		}
	}
	result.Insert(meta.Motion{"g", "d"}, gotoDetailViewCmd)

	// TODO: gg/G motion

	return result
}

func (sm *globalSearchModal) CommandSet() meta.Trie[tea.Msg] {
	return meta.Trie[tea.Msg]{}
}

func (sm *globalSearchModal) Reload() Modal {
	return newGlobalSearchModal(sm.DB)
}

func makeSelectAllDataCmd(DB *sqlx.DB) tea.Cmd {
	return func() tea.Msg {
		var result []any

		ledgers := database.AvailableLedgers()

		accounts := database.AvailableAccounts()

		journals := database.AvailableJournals()

		entries, err := database.SelectEntries(DB)
		if err != nil {
			return meta.MessageCmd(err)
		}

		entryRows, err := database.SelectRows(DB)
		if err != nil {
			return meta.MessageCmd(err)
		}

		result = slices.Concat(
			toAnySlice(ledgers),
			toAnySlice(accounts),
			toAnySlice(journals),
			toAnySlice(entries),
			toAnySlice(entryRows),
		)

		return meta.GlobalSearchDataLoadedMsg{Data: result}
	}
}

func toAnySlice[T any](slice []T) []any {
	result := make([]any, len(slice))

	for i, value := range slice {
		result[i] = searchItem{
			model: value,
		}
	}

	return result
}
