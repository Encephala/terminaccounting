package main

import (
	"context"
	"errors"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

func runSSHServer(DB *sqlx.DB, addr string) error {
	teaHandler := func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		ta := newTerminaccounting(DB)

		return ta, []tea.ProgramOption{tea.WithAltScreen()}
	}

	server, err := wish.NewServer(
		wish.WithAddress(addr),
		wish.WithHostKeyPath(".ssh/terminaccounting_ed25519"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	slog.Info("Starting SSH server", "addr", addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			slog.Error("SSH server error", "error", err)
		}
	}()

	<-ctx.Done()

	slog.Info("Stopping SSH server")

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	return server.Shutdown(shutdownCtx)
}
