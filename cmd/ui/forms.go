package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shopspring/decimal"
	"github.com/zlovtnik/gprint/cmd/ui/api"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

// Login form initialization
func (m Model) initLoginForm() (tea.Model, tea.Cmd) {
	m.inputs = make([]textinput.Model, 2)

	// Username input
	username := textinput.New()
	username.Placeholder = "Username"
	username.Focus()
	m.inputs[0] = username

	// Password input
	password := textinput.New()
	password.Placeholder = "Password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	m.inputs[1] = password

	m.focusIndex = 0
	m.view = ui.ViewLogin
	m.formEntity = "login"
	m.formAction = "login"

	return m, textinput.Blink
}

// Form initialization functions
func (m Model) initCustomerForm(customer *api.Customer) (tea.Model, tea.Cmd) {
	m.inputs = make([]textinput.Model, 6)

	fields := []struct {
		placeholder string
		value       string
	}{
		{"Customer Code", ""},
		{"Name", ""},
		{"Type (INDIVIDUAL/COMPANY)", "INDIVIDUAL"},
		{"Email", ""},
		{"Phone", ""},
		{labelTaxID, ""},
	}

	if customer != nil {
		fields[0].value = customer.CustomerCode
		fields[1].value = customer.Name
		fields[2].value = customer.CustomerType
		fields[3].value = customer.Email
		fields[4].value = customer.Phone
		fields[5].value = customer.TaxID
		m.view = ui.ViewCustomerEdit
		m.formAction = "edit"
	} else {
		m.view = ui.ViewCustomerCreate
		m.formAction = "create"
	}

	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.placeholder
		ti.SetValue(f.value)
		if i == 0 {
			ti.Focus()
		}
		m.inputs[i] = ti
	}

	m.focusIndex = 0
	m.formEntity = "customer"
	return m, textinput.Blink
}

func (m Model) initServiceForm(service *api.Service) (tea.Model, tea.Cmd) {
	m.inputs = make([]textinput.Model, 6)

	fields := []struct {
		placeholder string
		value       string
	}{
		{"Service Code", ""},
		{"Name", ""},
		{"Description", ""},
		{"Category", ""},
		{"Unit Price", "0.00"},
		{"Price Unit (HOUR/DAY/MONTH/PROJECT/UNIT)", "HOUR"},
	}

	if service != nil {
		fields[0].value = service.ServiceCode
		fields[1].value = service.Name
		fields[2].value = service.Description
		fields[3].value = service.Category
		fields[4].value = service.UnitPrice.StringFixed(2)
		fields[5].value = service.PriceUnit
		m.view = ui.ViewServiceEdit
		m.formAction = "edit"
	} else {
		m.view = ui.ViewServiceCreate
		m.formAction = "create"
	}

	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.placeholder
		ti.SetValue(f.value)
		if i == 0 {
			ti.Focus()
		}
		m.inputs[i] = ti
	}

	m.focusIndex = 0
	m.formEntity = "service"
	return m, textinput.Blink
}

func (m Model) initContractForm(contract *api.Contract) (tea.Model, tea.Cmd) {
	m.inputs = make([]textinput.Model, 5)

	fields := []struct {
		placeholder string
		value       string
	}{
		{"Contract Number", ""},
		{labelCustomerID, ""},
		{"Contract Type (SERVICE/RECURRING/PROJECT)", "SERVICE"},
		{"Billing Cycle (MONTHLY/QUARTERLY/YEARLY/ONCE)", "MONTHLY"},
		{labelTotalValue, "0.00"},
	}

	if contract != nil {
		fields[0].value = contract.ContractNumber
		fields[1].value = fmt.Sprintf("%d", contract.CustomerID)
		fields[2].value = contract.ContractType
		fields[3].value = contract.BillingCycle
		fields[4].value = contract.TotalValue.StringFixed(2)
		m.view = ui.ViewContractEdit
		m.formAction = "edit"
	} else {
		m.view = ui.ViewContractCreate
		m.formAction = "create"
	}

	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.placeholder
		ti.SetValue(f.value)
		if i == 0 {
			ti.Focus()
		}
		m.inputs[i] = ti
	}

	m.focusIndex = 0
	m.formEntity = "contract"
	return m, textinput.Blink
}

// Form submission handlers
func (m Model) handleCustomerFormSubmit() (tea.Model, tea.Cmd) {
	if m.formAction == "create" {
		req := &api.CreateCustomerRequest{
			CustomerCode: m.inputs[0].Value(),
			Name:         m.inputs[1].Value(),
			CustomerType: m.inputs[2].Value(),
			Email:        m.inputs[3].Value(),
			Phone:        m.inputs[4].Value(),
			TaxID:        m.inputs[5].Value(),
			Address:      map[string]string{},
		}
		return m, m.createCustomer(req)
	}
	req := &api.UpdateCustomerRequest{
		CustomerCode: m.inputs[0].Value(),
		Name:         m.inputs[1].Value(),
		CustomerType: m.inputs[2].Value(),
		Email:        m.inputs[3].Value(),
		Phone:        m.inputs[4].Value(),
		TaxID:        m.inputs[5].Value(),
		Address:      map[string]string{},
	}
	// Guard against nil selectedCustomer
	if m.selectedCustomer == nil {
		m.message = "No customer selected"
		m.messageType = ui.MessageTypeError
		return m, nil
	}
	return m, m.updateCustomer(m.selectedCustomer.ID, req)
}

func (m Model) handleServiceFormSubmit() (tea.Model, tea.Cmd) {
	priceDecimal, err := decimal.NewFromString(m.inputs[4].Value())
	if err != nil {
		m.message = "Invalid price format. Please enter a valid number."
		m.messageType = ui.MessageTypeError
		m.focusIndex = 4
		m = m.updateInputFocus()
		return m, nil
	}

	if m.formAction == "create" {
		req := &api.CreateServiceRequest{
			ServiceCode: m.inputs[0].Value(),
			Name:        m.inputs[1].Value(),
			Description: m.inputs[2].Value(),
			Category:    m.inputs[3].Value(),
			UnitPrice:   priceDecimal,
			PriceUnit:   m.inputs[5].Value(),
			Currency:    ui.DefaultCurrency,
		}
		return m, m.createService(req)
	}
	req := &api.UpdateServiceRequest{
		ServiceCode: m.inputs[0].Value(),
		Name:        m.inputs[1].Value(),
		Description: m.inputs[2].Value(),
		Category:    m.inputs[3].Value(),
		UnitPrice:   &priceDecimal,
		PriceUnit:   m.inputs[5].Value(),
		Currency:    ui.DefaultCurrency,
	}
	// Guard against nil selectedService
	if m.selectedService == nil {
		m.message = "No service selected"
		m.messageType = ui.MessageTypeError
		return m, nil
	}
	return m, m.updateService(m.selectedService.ID, req)
}

func (m Model) handleContractFormSubmit() (tea.Model, tea.Cmd) {
	customerID, err := strconv.ParseInt(m.inputs[1].Value(), 10, 64)
	if err != nil {
		m.message = "Invalid Customer ID. Please enter a valid number."
		m.messageType = ui.MessageTypeError
		m.focusIndex = 1
		m = m.updateInputFocus()
		return m, nil
	}

	// Validate TotalValue as a decimal
	totalValue, err := decimal.NewFromString(m.inputs[4].Value())
	if err != nil {
		m.message = "Invalid Total Value. Please enter a valid number."
		m.messageType = ui.MessageTypeError
		m.focusIndex = 4
		m = m.updateInputFocus()
		return m, nil
	}

	if m.formAction == "create" {
		req := &api.CreateContractRequest{
			ContractNumber: m.inputs[0].Value(),
			CustomerID:     customerID,
			ContractType:   m.inputs[2].Value(),
			BillingCycle:   m.inputs[3].Value(),
			TotalValue:     totalValue,
		}
		return m, m.createContract(req)
	}
	req := &api.UpdateContractRequest{
		ContractNumber: m.inputs[0].Value(),
		CustomerID:     &customerID,
		ContractType:   m.inputs[2].Value(),
		BillingCycle:   m.inputs[3].Value(),
		TotalValue:     &totalValue,
	}
	// Guard against nil selectedContract
	if m.selectedContract == nil {
		m.message = "No contract selected"
		m.messageType = ui.MessageTypeError
		return m, nil
	}
	return m, m.updateContract(m.selectedContract.ID, req)
}
