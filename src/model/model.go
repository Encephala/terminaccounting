package model

import (
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const LEADER = " "

type inputMode string

const NORMALMODE inputMode = "NORMAL"
const INSERTMODE inputMode = "INSERT"
const COMMANDMODE inputMode = "COMMAND"

type Model struct {
	Db *sqlx.DB

	ViewWidth, ViewHeight int

	ActiveApp int

	Apps []meta.App

	// current vim-esque input mode
	InputMode inputMode

	// vim-esque command input
	CommandInput textinput.Model

	// current vim-esque key stroke
	CurrentStroke []string
}
