# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All commands run from the `src/` directory.

```bash
go build -v          # Build the binary
go test ./...        # Run all tests
go test -run TestFoo # Run a specific test
go vet ./...         # Lint
```

## Architecture

Terminaccounting is a TUI personal finance app built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). The app has four tabs: Entries, Ledgers, Accounts, Journals — each is an `App` (see `meta/meta.go`).

### Model hierarchy

```
terminaccounting (tea.Model, main.go/model.go)
  └── appManager (appmanager.go)
        └── App (meta.App interface) — one of: EntriesApp, LedgersApp, AccountsApp, JournalsApp
              └── View (view/view.go interface) — list, detail, create, update, delete
```

Each layer implements `Init() / Update() / View()`. Apps use a "self-returning" `Update(msg) (App, tea.Cmd)` signature rather than `tea.Model`'s generic interface.

### Vim input mode system

Three modes live in `meta/vim.go`: `NORMALMODE`, `INSERTMODE`, `COMMANDMODE`. Mode-switching is handled in `model.go`. Key bindings are stored in **Trie** structures (`meta/trie.go`) — both motions and commands use prefix-tree lookup, enabling multi-key sequences (e.g. `gt`, `gT`). Each app exposes `CurrentMotionSet()` and `CurrentCommandSet()` for its active view.

### Message flow

All inter-component communication goes through `tea.Msg`. Custom message types are in `meta/messages.go`. Use `meta.MessageCmd(msg)` to wrap a value as a `tea.Cmd`. Global commands (`:quit`, `:refreshcache`, `:messages`) are intercepted in the top-level `terminaccounting.Update()`; view-specific commands are handled inside each app.

### View system (`view/`)

- **listview** — scrollable list with search
- **detailview** — single item with nested rows (e.g. entry line items / reconciliation)
- **mutateview** — shared generic form for both create and update
- **deleteview** — confirmation prompt

Form fields are wrapped by `inputAdapter` (polymorphic over `textinput`, `textarea`, `itempicker`, `booleaninput`). `inputManager` handles focus and tab-navigation across mixed inputs.

### Database layer (`database/`)

SQLite3 via `sqlx`. Schema is initialised by `database.InitSchemas()`. An in-memory cache of ledgers, accounts, and journals is populated on startup via `database.UpdateCache()` and refreshed with `:refreshcache`. Entries are fetched lazily.

### Test harness (`tat/`)

`tat.SetupTestEnv(t)` creates a per-test in-memory SQLite DB. `TestWrapper[T]` simulates the Bubble Tea runtime: `Send(msgs...)` dispatches messages and recursively runs returned commands; `SendText(s)` sends rune-by-rune key messages. Use `NewTestWrapperSpecific` for self-typed models (Apps/Views) and `NewTestWrapperGeneric` for plain `tea.Model`. Tests that produce messages only handled by the outer model pass those types as `ignoredMsgs`.

## Principles

You will only write tests. You may suggest changes needed to code that both improve the code and make testing more straightforward.
Be consistent in your test style.
Do not follow the Golang convention to have variables names just be individual letters - use verbose variable names as a rule, individual letters as an exception (e.g. for a temporary variable/if the name you want to use is already being used).

You will not create spurious test files. Group tests that are related into one long test file.
E.g. not `createview_test.go` and `entries_createview_test.go`, but just one long `createview_test.go`.

Don't mock stuff.
Keep things simple.

Use comments sparsely. Only add a comment when it gives context as to why the code is the way it is.

DO NOT add comments to explain how the code works.

Use sqlite in-memory databases for testing if needed.

Don't use `go build` to validate code. Use `go test`. I don't want to have stray build artefacts.
