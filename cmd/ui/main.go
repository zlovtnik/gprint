package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zlovtnik/gprint/cmd/ui/api"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

// UI string constants to avoid duplication
const (
	fmtRequestTimeout = "request timeout: %w"
	dateTimeFormat    = "2006-01-02 15:04"
	backToMainMenu    = "← Back to Main Menu"
	formSaveCancel    = "Press Enter to save, Esc to cancel"
	labelTaxID        = "Tax ID"
	labelCustomerID   = "Customer ID"
	labelTotalValue   = "Total Value"
	fmtMenuItemNL     = "%s%s\n"
	fmtMenuItemNL2    = "%s%s\n\n"
	fmtDetailRow      = "%s %s\n"
	fmtFormTitle      = "%s\n%s\n\n"
)

// Model is the main application model
type Model struct {
	client      *api.Client
	view        ui.ViewState
	cursor      int
	message     string
	messageType string // "error", "success", "info"

	// Data
	customers []api.Customer
	services  []api.Service
	contracts []api.Contract
	printJobs []api.PrintJob

	// Selected items
	selectedCustomer *api.Customer
	selectedService  *api.Service
	selectedContract *api.Contract
	selectedPrintJob *api.PrintJob

	// Form inputs
	inputs     []textinput.Model
	focusIndex int
	formAction string
	formEntity string

	// Settings / Auth
	baseURL  string
	token    string
	user     string
	tenantID string
	signer   string

	// UI state
	sidebarOpen    bool
	sidebarCursor  int
	focusOnSidebar bool

	// Window size
	width  int
	height int
}

func initialModel() Model {
	baseURL := os.Getenv("GPRINT_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	client, err := api.NewClient(baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid API URL %q: %v\n", baseURL, err)
		os.Exit(1)
	}

	// Check for token in environment
	token := os.Getenv("GPRINT_TOKEN")
	if token != "" {
		client.SetToken(token)
	}

	// Get signer from environment, default to "UI User"
	signer := os.Getenv("SIGNER_NAME")
	if signer == "" {
		signer = "UI User"
	}

	// Determine initial view: login if no token, main otherwise
	initialView := ui.ViewMain
	var inputs []textinput.Model

	if token == "" {
		initialView = ui.ViewLogin
		// Initialize login form inputs
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

	// Set formEntity only for login view
	formEntity := ""
	if initialView == ui.ViewLogin {
		formEntity = "login"
	}

	return Model{
		client:      client,
		view:        initialView,
		baseURL:     baseURL,
		token:       token,
		signer:      signer,
		sidebarOpen: true,
		width:       80,
		height:      24,
		inputs:      inputs,
		formEntity:  formEntity,
	}
}

func (m Model) Init() tea.Cmd {
	// If we already have a token, fetch all data on startup
	if m.token != "" {
		return tea.Batch(textinput.Blink, m.fetchAllData())
	}
	return textinput.Blink
}

// Messages for async operations
type fetchCustomersMsg struct{ customers []api.Customer }
type fetchServicesMsg struct{ services []api.Service }
type fetchContractsMsg struct{ contracts []api.Contract }
type fetchPrintJobsMsg struct{ jobs []api.PrintJob }
type errMsg struct{ err error }
type successMsg struct{ message string }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case fetchCustomersMsg:
		return m.handleFetchCustomers(msg), nil
	case fetchServicesMsg:
		return m.handleFetchServices(msg), nil
	case fetchContractsMsg:
		return m.handleFetchContracts(msg), nil
	case fetchPrintJobsMsg:
		return m.handleFetchPrintJobs(msg), nil
	case errMsg:
		return m.handleError(msg), nil
	case successMsg:
		return m.handleSuccess(msg), nil
	case loginMsg:
		return m.handleLoginMsgWithCmd(msg)
	}

	// Update text inputs if in form mode
	if len(m.inputs) > 0 {
		return m.updateInputs(msg)
	}

	return m, nil
}

// handleFetchCustomers processes customer fetch results
func (m Model) handleFetchCustomers(msg fetchCustomersMsg) Model {
	m.customers = msg.customers
	m.message = fmt.Sprintf("Loaded %d customers", len(msg.customers))
	m.messageType = "success"
	return m
}

// handleFetchServices processes service fetch results
func (m Model) handleFetchServices(msg fetchServicesMsg) Model {
	m.services = msg.services
	m.message = fmt.Sprintf("Loaded %d services", len(msg.services))
	m.messageType = "success"
	return m
}

// handleFetchContracts processes contract fetch results
func (m Model) handleFetchContracts(msg fetchContractsMsg) Model {
	m.contracts = msg.contracts
	m.message = fmt.Sprintf("Loaded %d contracts", len(msg.contracts))
	m.messageType = "success"
	return m
}

// handleFetchPrintJobs processes print job fetch results
func (m Model) handleFetchPrintJobs(msg fetchPrintJobsMsg) Model {
	m.printJobs = msg.jobs
	m.message = fmt.Sprintf("Loaded %d print jobs", len(msg.jobs))
	m.messageType = "success"
	return m
}

// handleError processes error messages
func (m Model) handleError(msg errMsg) Model {
	m.message = msg.err.Error()
	m.messageType = "error"
	return m
}

// handleSuccess processes success messages
func (m Model) handleSuccess(msg successMsg) Model {
	m.message = msg.message
	m.messageType = "success"
	return m
}

// handleLoginMsg processes login response
func (m Model) handleLoginMsg(msg loginMsg) Model {
	if msg.err != nil {
		m.message = msg.err.Error()
		m.messageType = "error"
		return m
	}
	if msg.resp == nil {
		m.message = "login failed: empty response from server"
		m.messageType = "error"
		return m
	}
	m.token = msg.resp.AccessToken
	// Update client with new token for future API calls
	m.client.SetToken(m.token)
	m.user = msg.resp.User
	m.tenantID = msg.resp.TenantID
	m.message = fmt.Sprintf("Welcome, %s!", msg.resp.User)
	m.messageType = "success"
	m.inputs = nil
	m.view = ui.ViewMain
	return m
}

// handleLoginMsgWithCmd processes login response and returns a command to fetch all data
func (m Model) handleLoginMsgWithCmd(msg loginMsg) (tea.Model, tea.Cmd) {
	m = m.handleLoginMsg(msg)
	// If login was successful, fetch all data
	if m.token != "" && m.view == ui.ViewMain {
		return m, m.fetchAllData()
	}
	return m, nil
}

// handleKeyMsg processes keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear message on any key except enter
	if m.message != "" && msg.String() != "enter" {
		m.message = ""
	}

	inFormMode := len(m.inputs) > 0

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m.handleQuitKey(msg, inFormMode)
	case "esc":
		return m.handleEscKey()
	case "up", "k":
		return m.handleUpKey(msg.String(), inFormMode)
	case "down", "j":
		return m.handleDownKey(msg.String(), inFormMode)
	case "enter":
		return m.handleEnterKey()
	case "tab":
		return m.handleTabKey(inFormMode, 1)
	case "shift+tab":
		return m.handleTabKey(inFormMode, -1)
	case "n", "e", "d", "r":
		// Only handle shortcuts when NOT in form mode - let form inputs receive these keys
		if !inFormMode {
			return m.handleShortcutKey(msg.String())
		}
	case "ctrl+b":
		m.sidebarOpen = !m.sidebarOpen
		return m, nil
	case "left", "h":
		return m.handleLeftKey(inFormMode)
	case "right", "l":
		return m.handleRightKey(inFormMode)
	}

	// Pass through to text input handling
	if inFormMode {
		return m.updateInputs(msg)
	}
	return m, nil
}

// handleQuitKey handles the 'q' key
func (m Model) handleQuitKey(msg tea.KeyMsg, inFormMode bool) (tea.Model, tea.Cmd) {
	if inFormMode {
		return m.updateInputs(msg)
	}
	if m.view == ui.ViewMain || m.view == ui.ViewLogin {
		return m, tea.Quit
	}
	m.view = ui.ViewMain
	m.cursor = 0
	return m, nil
}

// handleEscKey handles the escape key
func (m Model) handleEscKey() (tea.Model, tea.Cmd) {
	if m.view == ui.ViewLogin {
		return m, nil
	}
	return m.handleEscape()
}

// handleUpKey handles up/k keys
func (m Model) handleUpKey(key string, inFormMode bool) (tea.Model, tea.Cmd) {
	if inFormMode {
		if key == "up" {
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			return m.updateInputFocus(), nil
		}
		return m, nil // Let 'k' through for typing
	}
	if m.focusOnSidebar {
		m.sidebarCursor = m.handleUp()
	} else {
		m.cursor = m.handleUp()
	}
	return m, nil
}

// handleDownKey handles down/j keys
func (m Model) handleDownKey(key string, inFormMode bool) (tea.Model, tea.Cmd) {
	if inFormMode {
		if key == "down" {
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m.updateInputFocus(), nil
		}
		return m, nil // Let 'j' through for typing
	}
	if m.focusOnSidebar {
		m.sidebarCursor = m.handleDown()
	} else {
		m.cursor = m.handleDown()
	}
	return m, nil
}

// handleEnterKey handles the enter key
func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.focusOnSidebar {
		return m.handleSidebarSelect()
	}
	return m.handleEnter()
}

// handleTabKey handles tab/shift+tab navigation
func (m Model) handleTabKey(inFormMode bool, direction int) (tea.Model, tea.Cmd) {
	if !inFormMode {
		return m, nil
	}
	m.focusIndex += direction
	if m.focusIndex >= len(m.inputs) {
		m.focusIndex = 0
	} else if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) - 1
	}
	return m.updateInputFocus(), nil
}

// handleShortcutKey handles n/e/d/r shortcuts (only called when not in form mode)
func (m Model) handleShortcutKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "n":
		return m.handleCreate()
	case "e":
		return m.handleEdit()
	case "d":
		return m.handleDelete()
	case "r":
		return m.handleRefresh()
	}
	return m, nil
}

// handleLeftKey handles left/h keys for sidebar focus
func (m Model) handleLeftKey(inFormMode bool) (tea.Model, tea.Cmd) {
	if inFormMode {
		return m, nil
	}
	if m.sidebarOpen && !m.focusOnSidebar {
		m.focusOnSidebar = true
		m.sidebarCursor = m.getSidebarIndexForView()
	}
	return m, nil
}

// handleRightKey handles right/l keys for content focus
func (m Model) handleRightKey(inFormMode bool) (tea.Model, tea.Cmd) {
	if inFormMode {
		return m, nil
	}
	if m.focusOnSidebar {
		m.focusOnSidebar = false
	}
	return m, nil
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
