package server

import (
	"bytes"
	"io"
	"runtime"

	"charm.land/ssh"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

var (
	lineFeed = []byte{'\n'}
	index    = []byte(ansi.Index)
)

func colorProfile(sess ssh.Session) colorprofile.Profile {
	env := sess.Environ()
	if pty, _, ok := sess.Pty(); ok {
		env = append(env, "TERM="+pty.Term)
	}
	return colorprofile.Env(env)
}

func programOutput(sess ssh.Session) io.Writer {
	if runtime.GOOS == "windows" && sess.EmulatedPty() {
		return lineFeedWriter{w: sess}
	}
	return sess
}

// Bubble Tea emits bare line feeds on Windows hosts, which the emulated pty would turn into CRLF.
type lineFeedWriter struct {
	w io.Writer
}

func (lw lineFeedWriter) Write(p []byte) (int, error) {
	if _, err := lw.w.Write(bytes.ReplaceAll(p, lineFeed, index)); err != nil {
		return 0, err
	}
	return len(p), nil
}
