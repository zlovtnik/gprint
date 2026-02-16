package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/zlovtnik/gprint/cmd/ui/api"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

const (
	defaultSSHHost = "0.0.0.0"
	defaultSSHPort = "2222"
)

// sshMain runs the SSH server mode
func sshMain() {
	host := os.Getenv("SSH_HOST")
	if host == "" {
		host = defaultSSHHost
	}

	port := os.Getenv("SSH_PORT")
	if port == "" {
		port = defaultSSHPort
	}

	keyPath := os.Getenv("SSH_KEY_PATH")
	if keyPath == "" {
		keyPath = ".ssh/gprint_ed25519"
	}

	// Ensure .ssh directory exists
	if err := os.MkdirAll(".ssh", 0700); err != nil {
		log.Error("failed to create .ssh directory", "error", err)
		os.Exit(1)
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(keyPath),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			loggingMiddleware(),
		),
	)
	if err != nil {
		log.Error("failed to create SSH server", "error", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	log.Info("starting SSH server", "host", host, "port", port)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("SSH server error", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("shutting down SSH server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("failed to shutdown SSH server", "error", err)
	}
}

// teaHandler returns the Bubble Tea program for each SSH session
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, ok := s.Pty()

	// Default window dimensions for non-interactive sessions
	width, height := 80, 24
	if ok {
		width = pty.Window.Width
		height = pty.Window.Height
	}

	baseURL := os.Getenv("GPRINT_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	client, err := api.NewClient(baseURL)
	if err != nil {
		log.Error("failed to create API client", "error", err, "user", s.User())
		// Return a minimal error model
		return errorModel{err: fmt.Errorf("failed to connect to API: %w", err)}, nil
	}

	// Check for token in environment (server-side default)
	token := os.Getenv("GPRINT_TOKEN")
	if token != "" {
		client.SetToken(token)
	}

	signer := os.Getenv("SIGNER_NAME")
	if signer == "" {
		signer = s.User() // Use SSH username as signer
	}

	// Determine initial view
	initialView := ui.ViewMain
	var inputs []textinput.Model

	if token == "" {
		initialView = ui.ViewLogin
		inputs = make([]textinput.Model, 2)

		username := textinput.New()
		username.Placeholder = "Username"
		username.Focus()
		inputs[0] = username

		password := textinput.New()
		password.Placeholder = "Password"
		password.EchoMode = textinput.EchoPassword
		password.EchoCharacter = '•'
		inputs[1] = password
	}

	formEntity := ""
	if initialView == ui.ViewLogin {
		formEntity = "login"
	}

	m := Model{
		client:      client,
		view:        initialView,
		baseURL:     baseURL,
		token:       token,
		signer:      signer,
		sidebarOpen: true,
		width:       width,
		height:      height,
		inputs:      inputs,
		formEntity:  formEntity,
	}

	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

// errorModel is a minimal model for displaying connection errors
type errorModel struct {
	err error
}

func (m errorModel) Init() tea.Cmd { return nil }
func (m errorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		return m, tea.Quit
	}
	return m, nil
}
func (m errorModel) View() string {
	return fmt.Sprintf("\n  Error: %v\n\n  Press any key to exit.\n", m.err)
}

// loggingMiddleware logs SSH connections
func loggingMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			log.Info("SSH session started",
				"user", s.User(),
				"remote", s.RemoteAddr().String(),
			)
			next(s)
			log.Info("SSH session ended",
				"user", s.User(),
				"remote", s.RemoteAddr().String(),
			)
		}
	}
}
