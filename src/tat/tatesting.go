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
	_ "github.com/mattn/go-sqlite3"
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

	database.UpdateCache(DB)

	return DB
}

func sanitizeDsn(dsn string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(dsn, "_")
}

// A type similar to tea.Model, but instead of returning tea.Model in Update() it returns T (aka Self type)
type TeaModelEsque[T any] interface {
	Init() tea.Cmd
	Update(tea.Msg) (T, tea.Cmd)
	View() string
}

type TestWrapper[T any] struct {
	model T

	// Messages that should not be passed to the model
	// e.g. meta.NotificationMsg{} is only handled by the outer terminaccounting model
	ignoredMsgs []tea.Msg

	// The messages returned by the last tea.Cmd (recursively)
	LastCmdResults []tea.Msg
}

func (tw *TestWrapper[T]) init() {
	tw.Send(tea.WindowSizeMsg{Width: 100, Height: 40})

	switch model := any(tw.model).(type) {
	case TeaModelEsque[T]:
		tw.handleCmd(model.Init())

	case tea.Model:
		tw.handleCmd(model.Init())

	default:
		panic(fmt.Sprintf("unexpected type %#v", model))
	}
}

func (tw *TestWrapper[T]) update(msg tea.Msg) (T, tea.Cmd) {
	if tw.shouldIgnore(msg) {
		return tw.model, nil
	}

	switch model := any(tw.model).(type) {
	case TeaModelEsque[T]:
		return model.Update(msg)

	case tea.Model:
		newModel, cmd := model.Update(msg)
		return newModel.(T), cmd

	default:
		panic(fmt.Sprintf("unexpected type %#v", model))
	}
}

func (tw *TestWrapper[T]) shouldIgnore(msg tea.Msg) bool {
	for _, ignoredMsg := range tw.ignoredMsgs {
		if ignoredMsg == msg {
			return true
		}

		if err, ok := msg.(error); ok {
			if ignoredErr, ok := ignoredMsg.(error); ok && ignoredErr.Error() == err.Error() {
				return true
			}
		}
	}

	return false
}

func (tw *TestWrapper[T]) view() string {
	switch model := any(tw.model).(type) {
	case TeaModelEsque[T]:
		return model.View()

	case tea.Model:
		return model.View()

	default:
		panic(fmt.Sprintf("unexpected type %#v", model))
	}
}

func NewTestWrapperGeneric[T tea.Model](model T, ignoredMsgs ...tea.Msg) *TestWrapper[T] {
	tw := &TestWrapper[T]{model: model, ignoredMsgs: ignoredMsgs}

	tw.init()

	return tw
}

func NewTestWrapperSpecific[T TeaModelEsque[T]](model T, ignoredMsgs ...tea.Msg) *TestWrapper[T] {
	tw := &TestWrapper[T]{model: model, ignoredMsgs: ignoredMsgs}

	tw.init()

	return tw
}

func (tw *TestWrapper[T]) Send(messages ...tea.Msg) *TestWrapper[T] {
	tw.LastCmdResults = make([]tea.Msg, 0)

	for _, message := range messages {
		var cmd tea.Cmd
		tw.model, cmd = tw.update(message)

		tw.handleCmd(cmd)
	}

	return tw
}

func makeKeyMsg(input string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(input),
	}
}

func (tw *TestWrapper[T]) SendText(input string) *TestWrapper[T] {
	var messages []tea.Msg

	for _, char := range input {
		messages = append(messages, makeKeyMsg(string(char)))
	}

	tw.Send(messages...)

	return tw
}

// Simulate runtime handling cmds returned by an Update
func (tw *TestWrapper[T]) handleCmd(cmd tea.Cmd) {
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
			tw.model, cmd = tw.update(message)

			queue = append(queue, cmd)
		}
	}
}

func (tw *TestWrapper[T]) Execute(t *testing.T, function func(T)) {
	t.Helper()

	function(tw.model)
}

func (tw *TestWrapper[T]) AssertViewContains(t *testing.T, expected string) {
	t.Helper()

	tw.Execute(t, func(T) {
		assert.Contains(t, tw.view(), expected)
	})
}

func (tw *TestWrapper[T]) AssertLastMsgsEqual(t *testing.T, expected ...tea.Msg) {
	t.Helper()

	assert.Equal(t, expected, tw.LastCmdResults)
}
