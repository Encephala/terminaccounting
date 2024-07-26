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
	apps      []meta.App

	displayedError error
	fatalError     error // To print to screen on exit

	// current vim-esque input mode
	inputMode vim.InputMode
	// current motion
	currentMotion vim.Motion
	// known motions
	motions vim.Trie

	// vim-esque command input
	commandInput textinput.Model
}
