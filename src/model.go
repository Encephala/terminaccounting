package main

import (
	"terminaccounting/meta"
	"terminaccounting/vim"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	activeApp int

	apps []meta.App

	// current vim-esque input mode
	inputMode vim.InputMode

	// vim-esque command input
	commandInput textinput.Model

	// current vim-esque key stroke
	currentStroke vim.Stroke
}
