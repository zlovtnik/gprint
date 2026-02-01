package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

// Format string constants to avoid duplication
const (
	fmtCursorItem      = "%s%s\n"
	fmtCursorItemNL    = "%s%s\n\n"
	fmtKeyValue        = "%s %s\n"
	fmtLabelInput      = "%s\n%s\n\n"
	fmtDateTimeDisplay = "2006-01-02 15:04"
	msgFormSaveCancel  = "Press Enter to save, Esc to cancel"
	labelPrintJobs     = "Print Jobs"
)

// listConfig holds configuration for rendering a list view
type listConfig struct {
	title       string
	createLabel string // empty to disable create option
	itemCount   int
	cursor      int
	renderRow   func(idx int, selected bool) string
}

// renderList renders a standardized list view with optional create option and back button
func renderList(cfg listConfig) string {
	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render(cfg.title) + "\n\n")

	offset := 0

	// Create new option (if enabled)
	if cfg.createLabel != "" {
		cursor := "  "
		style := ui.MenuItemStyle
		if cfg.cursor == 0 {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf(fmtCursorItemNL, cursor, style.Render(cfg.createLabel)))
		offset = 1
	}

	// Item list
	for i := 0; i < cfg.itemCount; i++ {
		selected := cfg.cursor == i+offset
		b.WriteString(cfg.renderRow(i, selected))
	}

	// Back option
	b.WriteString("\n")
	backIdx := cfg.itemCount + offset
	cursor := "  "
	style := ui.MenuItemStyle
	if cfg.cursor == backIdx {
		cursor = ui.CursorStyle.Render("▸ ")
		style = ui.SelectedMenuItemStyle
	}
	b.WriteString(fmt.Sprintf(fmtCursorItem, cursor, style.Render("← Back to Main Menu")))

	return b.String()
}

// renderCursor returns cursor string and style based on selection state
func renderCursor(selected bool) (string, lipgloss.Style) {
	if selected {
		return ui.CursorStyle.Render("▸ "), ui.SelectedMenuItemStyle
	}
	return "  ", ui.MenuItemStyle
}

// View renders the entire UI using the new layout
func (m Model) View() string {
	// Login view is special - full screen, no layout
	if m.view == ui.ViewLogin {
		return m.renderLoginView()
	}
	return m.renderLayout()
}

// renderLoginView renders the full-screen login form
func (m Model) renderLoginView() string {
	// Center the login box
	boxWidth := 50

	var b strings.Builder

	// Cyberpunk ASCII Header - properly aligned
	b.WriteString(ui.HeaderTitleStyle.Render("╔════════════════════════════════════╗") + "\n")
	b.WriteString(ui.HeaderTitleStyle.Render("║") + ui.TitleStyle.Render("       ⚡ G P R I N T ⚡       ") + ui.HeaderTitleStyle.Render("║") + "\n")
	b.WriteString(ui.HeaderTitleStyle.Render("╚════════════════════════════════════╝") + "\n")
	b.WriteString(ui.SubtitleStyle.Render("   » Contract Printing Management «") + "\n\n")

	// Login form with neon border
	b.WriteString(ui.TitleStyle.Render("▓▓ LOGIN ▓▓") + "\n\n")

	if len(m.inputs) >= 2 {
		// Username field with neon label
		usernameLabel := ui.LabelStyle.Render("◈ Username")
		b.WriteString(usernameLabel + "\n")
		b.WriteString(m.inputs[0].View() + "\n\n")

		// Password field with neon label
		passwordLabel := ui.LabelStyle.Render("◈ Password")
		b.WriteString(passwordLabel + "\n")
		b.WriteString(m.inputs[1].View() + "\n\n")

		// Help text with neon separators
		help := ui.FooterKeyStyle.Render("Tab") + " " + ui.FooterLabelStyle.Render("Next Field") +
			ui.FooterHelpStyle.Render(" ║ ") +
			ui.FooterKeyStyle.Render("Enter") + " " + ui.FooterLabelStyle.Render("Login") +
			ui.FooterHelpStyle.Render(" ║ ") +
			ui.FooterKeyStyle.Render("Ctrl+C") + " " + ui.FooterLabelStyle.Render("Quit")
		b.WriteString(help + "\n")
	} else {
		b.WriteString(ui.InfoStyle.Render("◇ Loading login form...") + "\n")
	}

	// Error/success message
	if m.message != "" {
		var msgStyle lipgloss.Style
		switch m.messageType {
		case "error":
			msgStyle = ui.ErrorStyle
		case "success":
			msgStyle = ui.SuccessStyle
		default:
			msgStyle = ui.InfoStyle
		}
		b.WriteString("\n" + msgStyle.Render(m.message))
	}

	// Server info at bottom with neon styling
	b.WriteString("\n\n" + ui.FooterHelpStyle.Render("◇ Server: "+m.baseURL))

	// Center the box
	box := ui.BoxStyle.Width(boxWidth).Render(b.String())

	// Center horizontally and vertically
	boxHeight := strings.Count(box, "\n") + 1
	topPadding := (m.height - boxHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	leftPadding := (m.width - boxWidth - 4) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// Build centered view
	var result strings.Builder
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}
	for _, line := range strings.Split(box, "\n") {
		result.WriteString(strings.Repeat(" ", leftPadding) + line + "\n")
	}

	return result.String()
}

func (m Model) renderMainMenu() string {
	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render("Main Menu") + "\n\n")

	menuItems := ui.GetMainMenuItems()
	for i, item := range menuItems {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf("%s%s - %s\n", cursor, style.Render(item.Title), ui.SubtitleStyle.Render(item.Description)))
	}

	// Quit option
	cursor := "  "
	style := ui.MenuItemStyle
	if m.cursor == len(menuItems) {
		cursor = ui.CursorStyle.Render("▸ ")
		style = ui.SelectedMenuItemStyle
	}
	b.WriteString(fmt.Sprintf("\n%s%s\n", cursor, style.Render("Quit")))

	return b.String()
}

func (m Model) renderCustomerList() string {
	return renderList(listConfig{
		title:       "Customers",
		createLabel: "[+] Create New Customer",
		itemCount:   len(m.customers),
		cursor:      m.cursor,
		renderRow: func(idx int, selected bool) string {
			c := m.customers[idx]
			cursor, style := renderCursor(selected)
			status := ui.FormatBool(c.Active)
			return fmt.Sprintf("%s%s | %s | %s | %s\n",
				cursor,
				style.Render(fmt.Sprintf("%-10s", c.CustomerCode)),
				style.Render(fmt.Sprintf("%-30s", truncate(c.Name, 30))),
				c.CustomerType,
				status)
		},
	})
}

func (m Model) renderCustomerDetail() string {
	if m.selectedCustomer == nil {
		return "No customer selected"
	}
	c := m.selectedCustomer

	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render("Customer Details") + "\n\n")

	details := []struct {
		key   string
		value string
	}{
		{"ID", fmt.Sprintf("%d", c.ID)},
		{"Code", c.CustomerCode},
		{"Name", c.Name},
		{"Type", c.CustomerType},
		{"Trade Name", c.TradeName},
		{"Tax ID", c.TaxID},
		{"Email", c.Email},
		{"Phone", c.Phone},
		{"Active", ui.FormatBool(c.Active)},
		{"Created", c.CreatedAt.Format(fmtDateTimeDisplay)},
	}

	for _, d := range details {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.key+":"),
			ui.DetailValueStyle.Render(d.value)))
	}

	// Actions
	b.WriteString("\n" + ui.SubtitleStyle.Render("Actions") + "\n")
	actions := []string{"Edit", "Delete", "Back"}
	for i, action := range actions {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf(fmtCursorItem, cursor, style.Render(action)))
	}

	return b.String()
}

func (m Model) renderCustomerForm() string {
	var b strings.Builder
	title := "Create Customer"
	if m.formAction == "edit" {
		title = "Edit Customer"
	}
	b.WriteString(ui.SubtitleStyle.Render(title) + "\n\n")

	labels := []string{"Code", "Name", "Type", "Email", "Phone", "Tax ID"}
	// Warn if inputs and labels mismatch
	if len(m.inputs) != len(labels) {
		b.WriteString(ui.ErrorStyle.Render(fmt.Sprintf("[Form Error: expected %d inputs, got %d]\n\n", len(labels), len(m.inputs))))
	}
	// Use bounded iteration to prevent index out of range
	maxItems := len(labels)
	if len(m.inputs) < maxItems {
		maxItems = len(m.inputs)
	}
	for i := 0; i < maxItems; i++ {
		label := ui.LabelStyle.Render(labels[i] + ":")
		b.WriteString(fmt.Sprintf(fmtLabelInput, label, m.inputs[i].View()))
	}

	b.WriteString(ui.InfoStyle.Render(msgFormSaveCancel))
	return b.String()
}

func (m Model) renderServiceList() string {
	return renderList(listConfig{
		title:       "Services",
		createLabel: "[+] Create New Service",
		itemCount:   len(m.services),
		cursor:      m.cursor,
		renderRow: func(idx int, selected bool) string {
			s := m.services[idx]
			cursor, style := renderCursor(selected)
			status := ui.FormatBool(s.Active)
			return fmt.Sprintf("%s%s | %s | %s %s/%s | %s\n",
				cursor,
				style.Render(fmt.Sprintf("%-10s", s.ServiceCode)),
				style.Render(fmt.Sprintf("%-25s", truncate(s.Name, 25))),
				s.Currency,
				s.UnitPrice.StringFixed(2),
				s.PriceUnit,
				status)
		},
	})
}

func (m Model) renderServiceDetail() string {
	if m.selectedService == nil {
		return "No service selected"
	}
	s := m.selectedService

	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render("Service Details") + "\n\n")

	details := []struct {
		key   string
		value string
	}{
		{"ID", fmt.Sprintf("%d", s.ID)},
		{"Code", s.ServiceCode},
		{"Name", s.Name},
		{"Description", s.Description},
		{"Category", s.Category},
		{"Price", fmt.Sprintf("%s %s / %s", s.Currency, s.UnitPrice.StringFixed(2), s.PriceUnit)},
		{"Active", ui.FormatBool(s.Active)},
		{"Created", s.CreatedAt.Format(fmtDateTimeDisplay)},
	}

	for _, d := range details {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.key+":"),
			ui.DetailValueStyle.Render(d.value)))
	}

	// Actions
	b.WriteString("\n" + ui.SubtitleStyle.Render("Actions") + "\n")
	actions := []string{"Edit", "Delete", "Back"}
	for i, action := range actions {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf(fmtCursorItem, cursor, style.Render(action)))
	}

	return b.String()
}

func (m Model) renderServiceForm() string {
	var b strings.Builder
	title := "Create Service"
	if m.formAction == "edit" {
		title = "Edit Service"
	}
	b.WriteString(ui.SubtitleStyle.Render(title) + "\n\n")

	labels := []string{"Code", "Name", "Description", "Category", "Unit Price", "Price Unit"}
	// Warn if inputs and labels mismatch
	if len(m.inputs) != len(labels) {
		b.WriteString(ui.ErrorStyle.Render(fmt.Sprintf("[Form Error: expected %d inputs, got %d]\n\n", len(labels), len(m.inputs))))
	}
	// Use bounded iteration to prevent index out of range
	maxItems := len(labels)
	if len(m.inputs) < maxItems {
		maxItems = len(m.inputs)
	}
	for i := 0; i < maxItems; i++ {
		label := ui.LabelStyle.Render(labels[i] + ":")
		b.WriteString(fmt.Sprintf(fmtLabelInput, label, m.inputs[i].View()))
	}

	b.WriteString(ui.InfoStyle.Render(msgFormSaveCancel))
	return b.String()
}

func (m Model) renderContractList() string {
	return renderList(listConfig{
		title:       "Contracts",
		createLabel: "[+] Create New Contract",
		itemCount:   len(m.contracts),
		cursor:      m.cursor,
		renderRow: func(idx int, selected bool) string {
			c := m.contracts[idx]
			cursor, style := renderCursor(selected)
			status := ui.FormatStatus(c.Status)
			return fmt.Sprintf("%s%s | %s | %s | %s\n",
				cursor,
				style.Render(fmt.Sprintf("%-15s", c.ContractNumber)),
				c.ContractType,
				c.TotalValue.String(),
				status)
		},
	})
}

func (m Model) renderContractDetail() string {
	if m.selectedContract == nil {
		return "No contract selected"
	}
	c := m.selectedContract

	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render("Contract Details") + "\n\n")

	endDate := "N/A"
	if c.EndDate != nil {
		endDate = c.EndDate.Format("2006-01-02")
	}

	details := []struct {
		key   string
		value string
	}{
		{"ID", fmt.Sprintf("%d", c.ID)},
		{"Number", c.ContractNumber},
		{"Type", c.ContractType},
		{"Customer ID", fmt.Sprintf("%d", c.CustomerID)},
		{"Start Date", c.StartDate.Format("2006-01-02")},
		{"End Date", endDate},
		{"Total Value", c.TotalValue.String()},
		{"Billing Cycle", c.BillingCycle},
		{"Status", ui.FormatStatus(c.Status)},
		{"Created", c.CreatedAt.Format(fmtDateTimeDisplay)},
	}

	for _, d := range details {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.key+":"),
			ui.DetailValueStyle.Render(d.value)))
	}

	// Actions
	b.WriteString("\n" + ui.SubtitleStyle.Render("Actions") + "\n")
	actions := []string{"Edit", "Generate", "Print", "Sign", "Back"}
	for i, action := range actions {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf(fmtCursorItem, cursor, style.Render(action)))
	}

	return b.String()
}

func (m Model) renderContractForm() string {
	var b strings.Builder
	title := "Create Contract"
	if m.formAction == "edit" {
		title = "Edit Contract"
	}
	b.WriteString(ui.SubtitleStyle.Render(title) + "\n\n")

	labels := []string{"Contract Number", "Customer ID", "Type", "Billing Cycle", "Total Value"}
	// Warn if inputs and labels mismatch
	if len(m.inputs) != len(labels) {
		b.WriteString(ui.ErrorStyle.Render(fmt.Sprintf("[Form Error: expected %d inputs, got %d]\n\n", len(labels), len(m.inputs))))
	}
	// Use bounded iteration to prevent index out of range
	maxItems := len(labels)
	if len(m.inputs) < maxItems {
		maxItems = len(m.inputs)
	}
	for i := 0; i < maxItems; i++ {
		label := ui.LabelStyle.Render(labels[i] + ":")
		b.WriteString(fmt.Sprintf(fmtLabelInput, label, m.inputs[i].View()))
	}

	b.WriteString(ui.InfoStyle.Render(msgFormSaveCancel))
	return b.String()
}

func (m Model) renderPrintJobList() string {
	// Print jobs list has no create option and shows empty state
	if len(m.printJobs) == 0 {
		var b strings.Builder
		b.WriteString(ui.SubtitleStyle.Render(labelPrintJobs) + "\n\n")
		b.WriteString(ui.InfoStyle.Render("No print jobs found") + "\n\n")

		cursor, style := renderCursor(m.cursor == 0)
		b.WriteString(fmt.Sprintf(fmtMenuItemNL, cursor, style.Render(backToMainMenu)))
		return b.String()
	}

	return renderList(listConfig{
		title:       labelPrintJobs,
		createLabel: "", // no create option for print jobs
		itemCount:   len(m.printJobs),
		cursor:      m.cursor,
		renderRow: func(idx int, selected bool) string {
			j := m.printJobs[idx]
			cursor, style := renderCursor(selected)
			status := ui.FormatStatus(j.Status)
			return fmt.Sprintf("%s%s | Contract: %d | %s | %s\n",
				cursor,
				style.Render(fmt.Sprintf("#%-5d", j.ID)),
				j.ContractID,
				j.Format,
				status)
		},
	})
}

func (m Model) renderPrintJobDetail() string {
	if m.selectedPrintJob == nil {
		return "No print job selected"
	}
	j := m.selectedPrintJob

	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render("Print Job Details") + "\n\n")

	completedAt := "N/A"
	if j.CompletedAt != nil {
		completedAt = j.CompletedAt.Format(fmtDateTimeDisplay)
	}

	details := []struct {
		key   string
		value string
	}{
		{"ID", fmt.Sprintf("%d", j.ID)},
		{"Contract ID", fmt.Sprintf("%d", j.ContractID)},
		{"Format", j.Format},
		{"Status", ui.FormatStatus(j.Status)},
		{"File Size", fmt.Sprintf("%d bytes", j.FileSize)},
		{"Page Count", fmt.Sprintf("%d", j.PageCount)},
		{"Queued At", j.QueuedAt.Format(fmtDateTimeDisplay)},
		{"Completed At", completedAt},
		{"Requested By", j.RequestedBy},
	}

	for _, d := range details {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.key+":"),
			ui.DetailValueStyle.Render(d.value)))
	}

	b.WriteString("\n" + ui.InfoStyle.Render("Press Esc to go back"))
	return b.String()
}

func (m Model) renderSettings() string {
	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render("Settings") + "\n\n")

	details := []struct {
		key   string
		value string
	}{
		{"API URL", m.baseURL},
		{"Token", maskToken(m.token)},
	}

	for _, d := range details {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.key+":"),
			ui.DetailValueStyle.Render(d.value)))
	}

	b.WriteString("\n" + ui.InfoStyle.Render("Set GPRINT_API_URL and GPRINT_TOKEN environment variables"))
	return b.String()
}

// Helper functions
func truncate(s string, max int) string {
	if max <= 3 {
		if max <= 0 {
			return ""
		}
		runes := []rune(s)
		if len(runes) <= max {
			return s
		}
		return string(runes[:max])
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

func maskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) < 10 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
