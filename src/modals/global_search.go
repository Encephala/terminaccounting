package modals

import (
	"errors"
	"fmt"
	"slices"
	"terminaccounting/bubbles/list"
	"terminaccounting/database"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type globalSearchModal struct {
	DB *sqlx.DB

	width, height int

	list list.Model
}

func newGlobalSearchModal(DB *sqlx.DB) *globalSearchModal {
	return &globalSearchModal{
		DB: DB,

		list: list.New(0, 0),
	}
}

func (gsm *globalSearchModal) Init() tea.Cmd {
	return makeSelectAllDataCmd(gsm.DB)
}

func (gsm *globalSearchModal) Update(message tea.Msg) (Modal, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		gsm.width = message.Width
		gsm.height = message.Height

		var cmd tea.Cmd
		gsm.list, cmd = gsm.list.Update(message)

		return gsm, cmd

	case meta.NavigateMsg:
		gsm.list.Navigate(message.Direction == meta.DOWN)

		return gsm, nil

	case meta.JumpVerticalMsg:
		gsm.list.Jump(message.Down)

		return gsm, nil

	case meta.UpdateSearchMsg:
		var cmd tea.Cmd
		gsm.list, cmd = gsm.list.Update(list.FuzzyFilterMsg{Query: message.Query})

		return gsm, cmd

	case meta.DataLoadedMsg:
		gsm.list.SetItems(message.Data.([]list.Item))

		return gsm, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (gsm *globalSearchModal) View() string {
	return gsm.list.View()
}

func (gsm *globalSearchModal) AllowsInsertMode() bool {
	return false
}

func (gsm *globalSearchModal) AllowsSearchMode() bool {
	return true
}

func (gsm *globalSearchModal) MotionSet() meta.Trie[tea.Msg] {
	var result meta.Trie[tea.Msg]

	result.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	result.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})

	// Closures to the rescue once more
	var gotoDetailViewCmd tea.Cmd
	gotoDetailViewCmd = func() tea.Msg {
		activeItem := gsm.list.ActiveItem()

		if activeItem == nil {
			return errors.New("no items shown to go to detail view of")
		}

		var appType *meta.AppType
		var data any

		switch model := (*activeItem).(type) {
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

			entry, err := database.SelectEntry(gsm.DB, model.Entry)
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

	result.Insert(meta.Motion{"g", "g"}, meta.JumpVerticalMsg{Down: false})
	result.Insert(meta.Motion{"G"}, meta.JumpVerticalMsg{Down: true})

	return result
}

func (gsm *globalSearchModal) CommandSet() meta.Trie[tea.Msg] {
	return meta.Trie[tea.Msg]{}
}

func (gsm *globalSearchModal) Reload() Modal {
	return newGlobalSearchModal(gsm.DB)
}

func makeSelectAllDataCmd(DB *sqlx.DB) tea.Cmd {
	return func() tea.Msg {
		var result []list.Item

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
			toItemSlice(ledgers),
			toItemSlice(accounts),
			toItemSlice(journals),
			toItemSlice(entries),
			toItemSlice(entryRows),
		)

		return meta.DataLoadedMsg{Data: result}
	}
}

func toItemSlice[T list.Item](slice []T) []list.Item {
	result := make([]list.Item, len(slice))

	for i, value := range slice {
		result[i] = value
	}

	return result
}
