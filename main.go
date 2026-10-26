package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"charm.land/log/v2"
	"charm.land/ssh"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/NaiKiDEV/ssh-chat/internal/config"
	"github.com/NaiKiDEV/ssh-chat/internal/profile"
	"github.com/NaiKiDEV/ssh-chat/internal/server"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Error("ssh-chat stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		return err
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{ReportTimestamp: true, Level: cfg.LogLevel})

	profiles := profile.NewMemoryStore()
	if cfg.ProfilesPath != "" {
		if profiles, err = profile.Open(cfg.ProfilesPath); err != nil {
			return err
		}
	}

	hub, err := chat.NewHub(chat.Options{
		ListedRooms: cfg.Rooms,
		HistorySize: cfg.HistorySize,
		MaxRooms:    cfg.MaxRooms,
		MaxClients:  cfg.MaxUsers,
		NameOwner:   profiles.NameOwner,
	})
	if err != nil {
		return err
	}

	srv, err := server.New(server.Config{
		Addr:        cfg.Addr(),
		HostKeyPath: cfg.HostKeyPath,
		AllowGuests: cfg.AllowGuests,
		IdleTimeout: cfg.IdleTimeout,
	}, hub, profiles, logger)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe() }()
	logger.Info("listening", "addr", cfg.Addr(), "rooms", cfg.Rooms, "guests", cfg.AllowGuests)

	select {
	case err := <-serveErr:
		return fmt.Errorf("serving: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		return fmt.Errorf("shutting down: %w", err)
	}
	return nil
}
