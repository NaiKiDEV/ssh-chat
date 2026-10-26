package server

import (
	"net"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
)

const (
	maxConnsPerHost  = 10
	maxWindowWidth   = 500
	maxWindowHeight  = 300
	handshakeTimeout = 30 * time.Second
	maxConnLifetime  = 24 * time.Hour
	ipv6PrefixBits   = 64
)

type hostLimiter struct {
	max    int
	mu     sync.Mutex
	counts map[string]int
}

func newHostLimiter(max int) *hostLimiter {
	return &hostLimiter{max: max, counts: make(map[string]int)}
}

func (l *hostLimiter) wrap(_ ssh.Context, conn net.Conn) net.Conn {
	host := hostKey(conn.RemoteAddr())

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts[host] >= l.max {
		return nil
	}
	l.counts[host]++
	return &countedConn{Conn: conn, release: func() { l.release(host) }}
}

func (l *hostLimiter) release(host string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts[host]--; l.counts[host] <= 0 {
		delete(l.counts, host)
	}
}

type countedConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *countedConn) Close() error {
	c.once.Do(c.release)
	return c.Conn.Close()
}

func hostKey(addr net.Addr) string {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok {
		return addr.String()
	}
	if ip4 := tcp.IP.To4(); ip4 != nil {
		return ip4.String()
	}
	return tcp.IP.Mask(net.CIDRMask(ipv6PrefixBits, 128)).String()
}

func acceptPty(_ ssh.Context, pty ssh.Pty) bool {
	w, h := pty.Window.Width, pty.Window.Height
	return w > 0 && h > 0 && w <= maxWindowWidth && h <= maxWindowHeight
}

func filterMsg(_ tea.Model, msg tea.Msg) tea.Msg {
	switch msg := msg.(type) {
	case tea.SuspendMsg:
		return tea.ResumeMsg{}
	case tea.WindowSizeMsg:
		return tea.WindowSizeMsg{
			Width:  min(max(msg.Width, 1), maxWindowWidth),
			Height: min(max(msg.Height, 1), maxWindowHeight),
		}
	}
	return msg
}

func withLimits(hosts *hostLimiter) ssh.Option {
	return func(s *ssh.Server) error {
		s.PtyCallback = acceptPty
		s.ConnCallback = hosts.wrap
		s.HandshakeTimeout = handshakeTimeout
		s.MaxTimeout = maxConnLifetime
		return nil
	}
}
