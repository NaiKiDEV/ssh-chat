package server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	"charm.land/wish/v2/ratelimiter"
	"charm.land/wish/v2/recover"
	"github.com/NaiKiDEV/ssh-chat/internal/chat"
	"github.com/NaiKiDEV/ssh-chat/internal/profile"
	"github.com/NaiKiDEV/ssh-chat/internal/ui"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/time/rate"
)

const (
	rateLimiterEntries = 4096
	guestIDPrefix      = "guest:"
	defaultConnBurst   = 5
	shutdownGrace      = 2 * time.Second
)

var defaultConnRate = rate.Every(2 * time.Second)

type Config struct {
	Addr        string
	HostKeyPath string
	AllowGuests bool
	IdleTimeout time.Duration
	ConnRate    rate.Limit
	ConnBurst   int
}

type Server struct {
	hub      *chat.Hub
	profiles *profile.Store
	logger   *log.Logger
	ssh      *ssh.Server
}

func New(cfg Config, hub *chat.Hub, profiles *profile.Store, logger *log.Logger) (*Server, error) {
	s := &Server{hub: hub, profiles: profiles, logger: logger}

	if cfg.ConnRate <= 0 {
		cfg.ConnRate = defaultConnRate
	}
	if cfg.ConnBurst <= 0 {
		cfg.ConnBurst = defaultConnBurst
	}
	limiter := ratelimiter.NewRateLimiter(cfg.ConnRate, cfg.ConnBurst, rateLimiterEntries)
	panicLog := logger.StandardLog(log.StandardLogOptions{ForceLevel: log.ErrorLevel})

	opts := []ssh.Option{
		wish.WithAddress(cfg.Addr),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithPublicKeyAuth(func(ssh.Context, ssh.PublicKey) bool { return true }),
		wish.WithMiddleware(
			recover.MiddlewareWithLogger(panicLog,
				s.chatMiddleware(),
				activeterm.Middleware(),
				ratelimiter.Middleware(limiter),
			),
			logging.StructuredMiddlewareWithLogger(logger, log.InfoLevel),
		),
	}
	if cfg.IdleTimeout > 0 {
		opts = append(opts, wish.WithIdleTimeout(cfg.IdleTimeout))
	}
	if cfg.AllowGuests {
		opts = append(opts, wish.WithKeyboardInteractiveAuth(
			func(ssh.Context, gossh.KeyboardInteractiveChallenge) bool { return true },
		))
	}
	opts = append(opts, withLimits(newHostLimiter(maxConnsPerHost)))

	srv, err := wish.NewServer(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ssh server: %w", err)
	}
	s.ssh = srv
	return s, nil
}

func (s *Server) ListenAndServe() error {
	return s.ssh.ListenAndServe()
}

func (s *Server) Serve(ln net.Listener) error {
	return s.ssh.Serve(ln)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.hub.Close()
	graceCtx, cancel := context.WithTimeout(ctx, shutdownGrace)
	defer cancel()
	if err := s.ssh.Shutdown(graceCtx); !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return s.ssh.Close()
}

func (s *Server) chatMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			id := identityOf(sess)
			start, saved := s.initialProfile(id, sess.User())

			client, err := s.hub.Connect(id, start.Name, start.Color)
			if err != nil {
				wish.Fatalln(sess, err.Error())
				return
			}
			defer client.Close()

			acct := newAccount(client, s.profiles, s.logger, id, start, saved)
			app := bubbletea.MiddlewareWithProgramHandler(func(sess ssh.Session) *tea.Program {
				opts := append(bubbletea.MakeOptions(sess),
					tea.WithOutput(programOutput(sess)),
					tea.WithColorProfile(colorProfile(sess)),
					tea.WithFilter(filterMsg),
				)
				return tea.NewProgram(ui.New(client, acct), opts...)
			})
			app(next)(sess)

			if s.hub.Closed() {
				wish.Println(sess, "Server is shutting down, bye!")
			}
		}
	}
}

func (s *Server) initialProfile(id chat.Identity, user string) (profile.Profile, bool) {
	p, saved := profile.Profile{Name: user}, false
	if !id.Guest && s.profiles != nil {
		if stored, ok := s.profiles.Get(id.ID); ok {
			p, saved = stored, true
		}
	}
	if p.Color == "" {
		p.Color = profile.DefaultColor(id.ID)
	}
	return p, saved
}

func identityOf(sess ssh.Session) chat.Identity {
	key := sess.PublicKey()
	if key == nil {
		return chat.Identity{ID: guestIDPrefix + rand.Text(), Guest: true}
	}
	if cert, ok := key.(*gossh.Certificate); ok {
		return chat.Identity{ID: gossh.FingerprintSHA256(cert.Key)}
	}
	return chat.Identity{ID: gossh.FingerprintSHA256(key)}
}
