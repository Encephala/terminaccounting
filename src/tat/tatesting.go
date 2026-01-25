package tat

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"terminaccounting/database"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func SetupTestEnv(t *testing.T) *sqlx.DB {
	t.Helper()

	slog.SetLogLoggerLevel(slog.LevelWarn)

	// Ensure that each connection *within each test* uses the same in-memory database.
	id := fmt.Sprintf("%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	id = sanitizeDsn(id)

	DB := sqlx.MustConnect("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", id))
	t.Cleanup(func() { DB.Close() })

	err := database.InitSchemas(DB)

	require.Nil(t, err)

	return DB
}

func sanitizeDsn(dsn string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(dsn, "_")
}

func makeKeyMsg(input string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(input),
	}
}

type testWrapperBuilder struct {
	model tea.Model
}

func NewTestWrapperBuilder(model tea.Model) *testWrapperBuilder {
	return &testWrapperBuilder{model: model}
}

func (twb *testWrapperBuilder) RunSync(t *testing.T) *TestWrapper {
	t.Helper()

	tw := &TestWrapper{model: twb.model}

	tw.handleCmd(tw.model.Init())

	tw.Send(tea.WindowSizeMsg{Width: 100, Height: 40})

	return tw
}

type TestWrapper struct {
	model tea.Model

	LastCmdResults []tea.Msg
}

func (tw *TestWrapper) Send(messages ...tea.Msg) *TestWrapper {
	tw.LastCmdResults = make([]tea.Msg, 0)

	for _, message := range messages {
		var cmd tea.Cmd
		tw.model, cmd = tw.model.Update(message)

		tw.handleCmd(cmd)
	}

	return tw
}

func (tw *TestWrapper) SendText(input string) *TestWrapper {
	var messages []tea.Msg

	for _, char := range input {
		messages = append(messages, makeKeyMsg(string(char)))
	}

	tw.Send(messages...)

	return tw
}

// Simulate runtime handling cmds returned by an Update
func (tw *TestWrapper) handleCmd(cmd tea.Cmd) {
	var queue []tea.Cmd
	queue = append(queue, cmd)

	for len(queue) > 0 {
		cmd := queue[0]
		queue = queue[1:]

		if cmd == nil {
			continue
		}

		switch message := cmd().(type) {
		case tea.BatchMsg:
			queue = append(queue, message...)

		// Nil message, e.g. meta.MessageCmd(err) but err was nil
		case nil:
			continue

		default:
			tw.LastCmdResults = append(tw.LastCmdResults, message)
			tw.model, cmd = tw.model.Update(message)

			queue = append(queue, cmd)
		}
	}
}

func (tw *TestWrapper) Assert(t *testing.T, getter func(tea.Model) bool) {
	t.Helper()

	assert.True(t, getter(tw.model))
}

func (tw *TestWrapper) AssertViewContains(t *testing.T, expected string) {
	t.Helper()

	assert.Contains(t, tw.model.View(), expected)
}

func (tw *TestWrapper) AssertLastMsgsEqual(t *testing.T, expected ...tea.Msg) {
	t.Helper()

	assert.Equal(t, expected, tw.LastCmdResults)
}
