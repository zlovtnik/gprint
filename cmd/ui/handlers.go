package main

import (
	"context"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zlovtnik/gprint/cmd/ui/api"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

// Error variables
var errLoginCredentialsRequired = errors.New("username and password are required")

// loginMsg is sent when login completes
type loginMsg struct {
	resp *api.LoginResponse
	err  error
}

// SidebarItem represents an item in the sidebar menu
type SidebarItem struct {
	Icon  string
	Title string
	View  ui.ViewState
}

// getSidebarItems returns the sidebar menu items
func getSidebarItems() []SidebarItem {
	return []SidebarItem{
		{Icon: "🏠", Title: "Dashboard", View: ui.ViewMain},
		{Icon: "👥", Title: "Customers", View: ui.ViewCustomers},
		{Icon: "🛠", Title: "Services", View: ui.ViewServices},
		{Icon: "📄", Title: "Contracts", View: ui.ViewContracts},
		{Icon: "🖨", Title: labelPrintJobs, View: ui.ViewPrintJobs},
		{Icon: "⚙", Title: "Settings", View: ui.ViewSettings},
	}
}

// handleLogin attempts to authenticate with the API
func (m Model) handleLogin() tea.Cmd {
	username := m.inputs[0].Value()
	password := m.inputs[1].Value()

	if username == "" || password == "" {
		return func() tea.Msg {
			return loginMsg{err: errLoginCredentialsRequired}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		resp, err := m.client.Login(ctx, username, password)
		return loginMsg{resp: resp, err: err}
	}
}

// getParentView returns the parent sidebar view for a given view
func getParentView(v ui.ViewState) ui.ViewState {
	switch v {
	case ui.ViewCustomerDetail, ui.ViewCustomerCreate, ui.ViewCustomerEdit:
		return ui.ViewCustomers
	case ui.ViewServiceDetail, ui.ViewServiceCreate, ui.ViewServiceEdit:
		return ui.ViewServices
	case ui.ViewContractDetail, ui.ViewContractCreate, ui.ViewContractEdit:
		return ui.ViewContracts
	case ui.ViewPrintJobDetail:
		return ui.ViewPrintJobs
	default:
		return v
	}
}

// getSidebarIndexForView returns the sidebar index for the current view
func (m Model) getSidebarIndexForView() int {
	targetView := getParentView(m.view)
	items := getSidebarItems()
	for i, item := range items {
		if item.View == targetView {
			return i
		}
	}
	return 0
}

// handleSidebarSelect handles selection from sidebar
func (m Model) handleSidebarSelect() (tea.Model, tea.Cmd) {
	items := getSidebarItems()
	if m.sidebarCursor >= 0 && m.sidebarCursor < len(items) {
		selectedView := items[m.sidebarCursor].View
		m.view = selectedView
		m.cursor = 0
		m.focusOnSidebar = false

		// Fetch data for the new view
		switch selectedView {
		case ui.ViewCustomers:
			return m, m.fetchCustomers()
		case ui.ViewServices:
			return m, m.fetchServices()
		case ui.ViewContracts:
			return m, m.fetchContracts()
		case ui.ViewPrintJobs:
			return m, m.fetchPrintJobs()
		}
	}
	return m, nil
}

// getBreadcrumb returns the breadcrumb path for current view
func (m Model) getBreadcrumb() []string {
	switch m.view {
	case ui.ViewLogin:
		return []string{"Login"}
	case ui.ViewMain:
		return []string{"Dashboard"}
	case ui.ViewCustomers:
		return []string{"Dashboard", "Customers"}
	case ui.ViewCustomerDetail:
		if m.selectedCustomer != nil {
			return []string{"Dashboard", "Customers", m.selectedCustomer.Name}
		}
		return []string{"Dashboard", "Customers", "Detail"}
	case ui.ViewCustomerCreate:
		return []string{"Dashboard", "Customers", "New Customer"}
	case ui.ViewCustomerEdit:
		return []string{"Dashboard", "Customers", "Edit"}
	case ui.ViewServices:
		return []string{"Dashboard", "Services"}
	case ui.ViewServiceDetail:
		if m.selectedService != nil {
			return []string{"Dashboard", "Services", m.selectedService.Name}
		}
		return []string{"Dashboard", "Services", "Detail"}
	case ui.ViewServiceCreate:
		return []string{"Dashboard", "Services", "New Service"}
	case ui.ViewServiceEdit:
		return []string{"Dashboard", "Services", "Edit"}
	case ui.ViewContracts:
		return []string{"Dashboard", "Contracts"}
	case ui.ViewContractDetail:
		if m.selectedContract != nil {
			return []string{"Dashboard", "Contracts", m.selectedContract.ContractNumber}
		}
		return []string{"Dashboard", "Contracts", "Detail"}
	case ui.ViewContractCreate:
		return []string{"Dashboard", "Contracts", "New Contract"}
	case ui.ViewContractEdit:
		return []string{"Dashboard", "Contracts", "Edit"}
	case ui.ViewPrintJobs:
		return []string{"Dashboard", labelPrintJobs}
	case ui.ViewPrintJobDetail:
		if m.selectedPrintJob != nil {
			return []string{"Dashboard", labelPrintJobs, fmt.Sprintf("Job #%d", m.selectedPrintJob.ID)}
		}
		return []string{"Dashboard", labelPrintJobs, "Detail"}
	case ui.ViewSettings:
		return []string{"Dashboard", "Settings"}
	default:
		return []string{"Dashboard"}
	}
}

func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	// Cannot escape from login if not authenticated
	if m.view == ui.ViewLogin {
		return m, nil
	}

	// If focused on sidebar, unfocus
	if m.focusOnSidebar {
		m.focusOnSidebar = false
		return m, nil
	}

	switch m.view {
	case ui.ViewCustomerDetail, ui.ViewCustomerCreate, ui.ViewCustomerEdit:
		m.view = ui.ViewCustomers
	case ui.ViewServiceDetail, ui.ViewServiceCreate, ui.ViewServiceEdit:
		m.view = ui.ViewServices
	case ui.ViewContractDetail, ui.ViewContractCreate, ui.ViewContractEdit:
		m.view = ui.ViewContracts
	case ui.ViewPrintJobDetail:
		m.view = ui.ViewPrintJobs
	default:
		m.view = ui.ViewMain
	}
	m.cursor = 0
	m.inputs = nil
	return m, nil
}

func (m Model) handleDown() int {
	if m.focusOnSidebar {
		items := getSidebarItems()
		if m.sidebarCursor < len(items)-1 {
			return m.sidebarCursor + 1
		}
		return m.sidebarCursor
	}
	maxItems := m.getMaxItems()
	if m.cursor < maxItems-1 {
		return m.cursor + 1
	}
	return m.cursor
}

func (m Model) handleUp() int {
	if m.focusOnSidebar {
		if m.sidebarCursor > 0 {
			return m.sidebarCursor - 1
		}
		return m.sidebarCursor
	}
	if m.cursor > 0 {
		return m.cursor - 1
	}
	return m.cursor
}

func (m Model) getMaxItems() int {
	switch m.view {
	case ui.ViewMain:
		return len(ui.GetMainMenuItems()) + 1 // +1 for Quit
	case ui.ViewCustomers:
		return len(m.customers) + 2 // +2 for Create and Back
	case ui.ViewServices:
		return len(m.services) + 2
	case ui.ViewContracts:
		return len(m.contracts) + 2
	case ui.ViewPrintJobs:
		return len(m.printJobs) + 1 // +1 for Back
	case ui.ViewCustomerDetail, ui.ViewServiceDetail:
		return 3 // Edit, Delete, Back
	case ui.ViewContractDetail:
		return 5 // Edit, Generate, Print, Sign, Back
	case ui.ViewPrintJobDetail:
		return 1 // No actions, just info display
	case ui.ViewCustomerCreate, ui.ViewCustomerEdit,
		ui.ViewServiceCreate, ui.ViewServiceEdit,
		ui.ViewContractCreate, ui.ViewContractEdit:
		return 1 // Safe minimum for form views
	case ui.ViewSettings, ui.ViewLogin:
		return 1 // Safe minimum for other views
	default:
		return 1 // Safe minimum for unknown views
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.view {
	case ui.ViewLogin:
		return m, m.handleLogin()
	case ui.ViewMain:
		return m.handleMainMenuSelect()
	case ui.ViewCustomers:
		return m.handleCustomerSelect()
	case ui.ViewServices:
		return m.handleServiceSelect()
	case ui.ViewContracts:
		return m.handleContractSelect()
	case ui.ViewPrintJobs:
		return m.handlePrintJobSelect()
	case ui.ViewCustomerCreate, ui.ViewCustomerEdit:
		return m.handleCustomerFormSubmit()
	case ui.ViewServiceCreate, ui.ViewServiceEdit:
		return m.handleServiceFormSubmit()
	case ui.ViewContractCreate, ui.ViewContractEdit:
		return m.handleContractFormSubmit()
	case ui.ViewCustomerDetail:
		return m.handleCustomerDetailAction()
	case ui.ViewServiceDetail:
		return m.handleServiceDetailAction()
	case ui.ViewContractDetail:
		return m.handleContractDetailAction()
	}
	return m, nil
}

func (m Model) handleMainMenuSelect() (tea.Model, tea.Cmd) {
	menuItems := ui.GetMainMenuItems()
	if m.cursor == len(menuItems) {
		return m, tea.Quit
	}
	// Bounds check to prevent panic
	if m.cursor < 0 || m.cursor >= len(menuItems) {
		m.cursor = 0
		return m, nil
	}
	item := menuItems[m.cursor]
	m.view = item.View
	m.cursor = 0

	switch item.View {
	case ui.ViewCustomers:
		return m, m.fetchCustomers()
	case ui.ViewServices:
		return m, m.fetchServices()
	case ui.ViewContracts:
		return m, m.fetchContracts()
	case ui.ViewPrintJobs:
		return m, m.fetchPrintJobs()
	case ui.ViewSettings:
		return m, nil
	}
	return m, nil
}

func (m Model) handleCustomerSelect() (tea.Model, tea.Cmd) {
	if m.cursor == 0 {
		return m.initCustomerForm(nil)
	}
	if m.cursor == len(m.customers)+1 {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	// Bounds check to prevent panic
	idx := m.cursor - 1
	if idx < 0 || idx >= len(m.customers) {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	cust := m.customers[idx]
	m.selectedCustomer = &cust
	m.view = ui.ViewCustomerDetail
	m.cursor = 0
	return m, nil
}

func (m Model) handleServiceSelect() (tea.Model, tea.Cmd) {
	if m.cursor == 0 {
		return m.initServiceForm(nil)
	}
	if m.cursor == len(m.services)+1 {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	// Bounds check to prevent panic
	idx := m.cursor - 1
	if idx < 0 || idx >= len(m.services) {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	svc := m.services[idx]
	m.selectedService = &svc
	m.view = ui.ViewServiceDetail
	m.cursor = 0
	return m, nil
}

func (m Model) handleContractSelect() (tea.Model, tea.Cmd) {
	if m.cursor == 0 {
		return m.initContractForm(nil)
	}
	if m.cursor == len(m.contracts)+1 {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	// Bounds check to prevent panic
	idx := m.cursor - 1
	if idx < 0 || idx >= len(m.contracts) {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	contr := m.contracts[idx]
	m.selectedContract = &contr
	m.view = ui.ViewContractDetail
	m.cursor = 0
	return m, nil
}

func (m Model) handlePrintJobSelect() (tea.Model, tea.Cmd) {
	if m.cursor == len(m.printJobs) {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	// Bounds check to prevent panic
	if m.cursor < 0 || m.cursor >= len(m.printJobs) {
		m.view = ui.ViewMain
		m.cursor = 0
		return m, nil
	}
	job := m.printJobs[m.cursor]
	m.selectedPrintJob = &job
	m.view = ui.ViewPrintJobDetail
	m.cursor = 0
	return m, nil
}

func (m Model) handleCreate() (tea.Model, tea.Cmd) {
	switch m.view {
	case ui.ViewCustomers:
		return m.initCustomerForm(nil)
	case ui.ViewServices:
		return m.initServiceForm(nil)
	case ui.ViewContracts:
		return m.initContractForm(nil)
	}
	return m, nil
}

func (m Model) handleEdit() (tea.Model, tea.Cmd) {
	switch m.view {
	case ui.ViewCustomerDetail:
		return m.initCustomerForm(m.selectedCustomer)
	case ui.ViewServiceDetail:
		return m.initServiceForm(m.selectedService)
	case ui.ViewContractDetail:
		return m.initContractForm(m.selectedContract)
	}
	return m, nil
}

func (m Model) handleDelete() (tea.Model, tea.Cmd) {
	switch m.view {
	case ui.ViewCustomerDetail:
		if m.selectedCustomer != nil {
			return m, m.deleteCustomer(m.selectedCustomer.ID)
		}
	case ui.ViewServiceDetail:
		if m.selectedService != nil {
			return m, m.deleteService(m.selectedService.ID)
		}
	}
	return m, nil
}

func (m Model) handleRefresh() (tea.Model, tea.Cmd) {
	switch m.view {
	case ui.ViewCustomers:
		return m, m.fetchCustomers()
	case ui.ViewServices:
		return m, m.fetchServices()
	case ui.ViewContracts:
		return m, m.fetchContracts()
	case ui.ViewPrintJobs:
		return m, m.fetchPrintJobs()
	}
	return m, nil
}

func (m Model) handleCustomerDetailAction() (tea.Model, tea.Cmd) {
	// Guard against nil selectedCustomer
	if m.selectedCustomer == nil {
		m.view = ui.ViewCustomers
		m.cursor = 0
		return m, nil
	}

	actions := []string{"Edit", "Delete", "Back"}
	if m.cursor < 0 || m.cursor >= len(actions) {
		return m, nil
	}

	switch actions[m.cursor] {
	case "Edit":
		return m.initCustomerForm(m.selectedCustomer)
	case "Delete":
		id := m.selectedCustomer.ID
		m.view = ui.ViewCustomers
		m.cursor = 0
		m.selectedCustomer = nil
		return m, m.deleteCustomer(id)
	case "Back":
		m.view = ui.ViewCustomers
		m.cursor = 0
	}
	return m, nil
}

func (m Model) handleServiceDetailAction() (tea.Model, tea.Cmd) {
	// Guard against nil selectedService
	if m.selectedService == nil {
		m.view = ui.ViewServices
		m.cursor = 0
		return m, nil
	}

	actions := []string{"Edit", "Delete", "Back"}
	if m.cursor >= 0 && m.cursor < len(actions) {
		switch actions[m.cursor] {
		case "Edit":
			return m.initServiceForm(m.selectedService)
		case "Delete":
			id := m.selectedService.ID
			m.view = ui.ViewServices
			m.cursor = 0
			m.selectedService = nil
			return m, m.deleteService(id)
		case "Back":
			m.view = ui.ViewServices
			m.cursor = 0
		}
	}
	return m, nil
}

func (m Model) handleContractDetailAction() (tea.Model, tea.Cmd) {
	// Guard against nil selectedContract
	if m.selectedContract == nil {
		m.view = ui.ViewContracts
		m.cursor = 0
		return m, nil
	}

	actions := []string{"Edit", "Generate", "Print", "Sign", "Back"}
	if m.cursor < 0 || m.cursor >= len(actions) {
		return m, nil
	}

	switch actions[m.cursor] {
	case "Edit":
		return m.initContractForm(m.selectedContract)
	case "Generate":
		return m, m.generateContract(m.selectedContract.ID)
	case "Print":
		return m, m.createPrintJob(m.selectedContract.ID, "PDF")
	case "Sign":
		return m, m.signContract(m.selectedContract.ID)
	case "Back":
		m.view = ui.ViewContracts
		m.cursor = 0
	}
	return m, nil
}

// updateInputFocus updates which form input is focused
func (m Model) updateInputFocus() Model {
	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return m
}

func (m Model) updateInputs(msg tea.Msg) (Model, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return m, tea.Batch(cmds...)
}
