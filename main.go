package main

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/NaiKiDEV/ssh-chat/internal/model"
	"github.com/NaiKiDEV/ssh-chat/internal/styles"
	"github.com/NaiKiDEV/ssh-chat/internal/terminal"
	"github.com/NaiKiDEV/ssh-chat/views/chat"
	"github.com/NaiKiDEV/ssh-chat/views/login"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const (
	host = "localhost"
	port = "23234"
)

const (
	VIEW_LOGIN = "login"
	VIEW_CHAT  = "chat"
)

type room struct {
	mutex sync.Mutex

	roomId      string
	messages    []model.Message
	activeUsers []string
}

func (r *room) AddMessage(msg model.Message) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.messages = append(r.messages, msg)
}

func (r *room) RemoveActiveUser(userName string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	activeUsers := r.activeUsers
	userIdx := slices.Index(activeUsers, userName)
	if userIdx != -1 {
		r.activeUsers[userIdx] = activeUsers[len(activeUsers)-1]
		r.activeUsers = activeUsers[:len(activeUsers)-1]
	}
}

func (r *room) AddActiveUser(userName string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.activeUsers = append(r.activeUsers, userName)
}

type ServerState struct {
	rooms map[string]*room
}

type user struct {
	displayName string
}

type clientState struct {
	terminalState *terminal.TerminalState
	clientStyles  *styles.ClientStyles
	renderer      *lipgloss.Renderer
	loginState    login.LoginState
	chatState     chat.ChatState
	user          *user
	activeView    string
	roomId        string
}

// Global var :(
var serverState ServerState

func main() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)

	serverState = ServerState{
		rooms: map[string]*room{
			"secret": {
				roomId:      "secret",
				messages:    []model.Message{},
				activeUsers: []string{},
			},
			"public": {
				roomId:      "public",
				messages:    []model.Message{},
				activeUsers: []string{},
			},
		},
	}

	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("Could not stop server", "error", err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()

	userName := s.User()

	renderer := bubbletea.MakeRenderer(s)
	cStyles := styles.NewClientStyles(renderer)

	theme := "light"
	if renderer.HasDarkBackground() {
		theme = "dark"
	}

	tState := &terminal.TerminalState{
		Width:  pty.Window.Width,
		Height: pty.Window.Height,
		Theme:  theme,
	}

	loginState := login.NewLoginState(userName)
	chatState := chat.NewChatState(userName, tState, cStyles)

	m := clientState{
		terminalState: tState,
		clientStyles:  cStyles,
		renderer:      renderer,
		loginState:    loginState,
		chatState:     chatState,
		activeView:    VIEW_LOGIN,
		// This should become available after login
		user: &user{
			displayName: userName,
		},
	}
	return m, []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}
}

func (m clientState) Init() tea.Cmd {
	return nil
}

func (m clientState) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalState.Height = msg.Height
		m.terminalState.Width = msg.Width

		if m.activeView == VIEW_CHAT {
			m.chatState = m.chatState.Resize(m.terminalState)
		}

	case tea.MouseMsg:
		if m.activeView == VIEW_CHAT {
			var cmd tea.Cmd
			m.chatState, cmd = m.chatState.Update(msg)
			return m, cmd
		}

	// Global input handling that takes priority over views
	case tea.KeyMsg:
		switch key := msg.Type; key {
		case tea.KeyCtrlC:
			serverRoom := serverState.rooms[m.roomId]
			if serverRoom == nil {
				return m, tea.Quit
			}

			// TODO: Should also handle non-graceful logouts
			serverRoom.RemoveActiveUser(m.user.displayName)

			return m, tea.Quit
		}

	case chat.MessageSentMsg:
		serverRoom := serverState.rooms[m.roomId]
		if serverRoom != nil {
			serverRoom.AddMessage(model.Message{
				Username:  m.user.displayName,
				Text:      msg.Message,
				Timestamp: time.Now(),
			})
		}
		return m, nil

	case login.RoomJoinRequestedMsg:
		roomId := msg.RoomId
		serverRoom := serverState.rooms[roomId]
		if serverRoom != nil {
			m.activeView = VIEW_CHAT
			m.chatState = m.chatState.SetRoom(serverRoom.roomId, serverRoom.messages)

			m.roomId = roomId
			serverRoom.AddActiveUser(m.user.displayName)
		} else {
			m.loginState = m.loginState.SetFormError("room not found")
			return m, nil
		}

	case chat.LeaveChatMsg:
		serverRoom := serverState.rooms[m.roomId]
		if serverRoom == nil {
			return m, tea.Quit
		}
		serverRoom.RemoveActiveUser(m.user.displayName)
		m.activeView = VIEW_LOGIN
		m.loginState = m.loginState.Reset()
		return m, nil

	}

	if m.activeView == VIEW_LOGIN {
		var cmd tea.Cmd
		m.loginState, cmd = m.loginState.Update(msg)
		return m, cmd
	}

	if m.activeView == VIEW_CHAT {
		var cmd tea.Cmd

		// TODO: maybe there is a better way?
		serverRoom := serverState.rooms[m.roomId]
		if serverRoom == nil {
			return m, nil
		}

		m.chatState = m.chatState.SetChatState(serverRoom.messages, serverRoom.activeUsers)
		m.chatState, cmd = m.chatState.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m clientState) View() string {
	var view strings.Builder

	if m.activeView == VIEW_LOGIN {
		view.WriteString(m.loginState.Render(m.terminalState, m.clientStyles))
	}

	if m.activeView == VIEW_CHAT {
		serverRoom := serverState.rooms[m.roomId]

		var activeUsers []string
		var messages []model.Message
		if serverRoom != nil {
			activeUsers = serverRoom.activeUsers
			messages = serverRoom.messages
		}

		view.WriteString(m.chatState.Render(m.terminalState, messages, activeUsers))
	}

	return view.String()
}
