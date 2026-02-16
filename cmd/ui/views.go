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

	// Card header
	header := ui.RenderCardHeader("◆", c.Name)

	// Build sections
	sections := []ui.CardSection{
		{
			Title: "Identity",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Code", Value: c.CustomerCode},
				{Label: "Type", Value: c.CustomerType},
				{Label: "Tax ID", Value: c.TaxID},
				{Label: "Trade Name", Value: c.TradeName},
			},
		},
		{
			Title: "Contact",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Email", Value: c.Email},
				{Label: "Phone", Value: c.Phone},
			},
		},
		{
			Title: "Status",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Active", Value: ui.FormatBool(c.Active)},
				{Label: "Created", Value: c.CreatedAt.Format(fmtDateTimeDisplay)},
				{Label: "ID", Value: fmt.Sprintf("%d", c.ID)},
			},
		},
	}

	cardWidth := 52
	b.WriteString(ui.RenderCard(header, sections, cardWidth))
	b.WriteString("\n")

	// Actions with icons
	b.WriteString(ui.CardSectionStyle.Render("⚡ Actions") + "\n")
	actions := []struct {
		icon string
		text string
	}{
		{"✎", "Edit"},
		{"✕", "Delete"},
		{"←", "Back"},
	}
	for i, action := range actions {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, action.icon, style.Render(action.text)))
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

	// Card header with service name
	header := ui.RenderCardHeader("◆", s.Name)

	// Build sections
	sections := []ui.CardSection{
		{
			Title: "Service Info",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Code", Value: s.ServiceCode},
				{Label: "Category", Value: s.Category},
				{Label: "Description", Value: truncate(s.Description, 35)},
			},
		},
		{
			Title: "Pricing",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Unit Price", Value: s.Currency + " " + s.UnitPrice.StringFixed(2)},
				{Label: "Price Unit", Value: s.PriceUnit},
			},
		},
		{
			Title: "Status",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Active", Value: ui.FormatBool(s.Active)},
				{Label: "Created", Value: s.CreatedAt.Format(fmtDateTimeDisplay)},
				{Label: "ID", Value: fmt.Sprintf("%d", s.ID)},
			},
		},
	}

	cardWidth := 52
	b.WriteString(ui.RenderCard(header, sections, cardWidth))
	b.WriteString("\n")

	// Actions with icons
	b.WriteString(ui.CardSectionStyle.Render("⚡ Actions") + "\n")
	actions := []struct {
		icon string
		text string
	}{
		{"✎", "Edit"},
		{"✕", "Delete"},
		{"←", "Back"},
	}
	for i, action := range actions {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, action.icon, style.Render(action.text)))
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

	// Card header with contract number and status badge
	header := ui.RenderCardHeader("◆", c.ContractNumber+" "+ui.FormatStatus(c.Status))

	endDate := "N/A"
	if c.EndDate != nil {
		endDate = c.EndDate.Format("2006-01-02")
	}

	// Build sections
	sections := []ui.CardSection{
		{
			Title: "Contract Info",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Type", Value: c.ContractType},
				{Label: "Customer ID", Value: fmt.Sprintf("%d", c.CustomerID)},
				{Label: "Billing Cycle", Value: c.BillingCycle},
			},
		},
		{
			Title: "Timeline",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Start Date", Value: c.StartDate.Format("2006-01-02")},
				{Label: "End Date", Value: endDate},
				{Label: "Created", Value: c.CreatedAt.Format(fmtDateTimeDisplay)},
			},
		},
		{
			Title: "Financial",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Total Value", Value: c.TotalValue.String()},
				{Label: "ID", Value: fmt.Sprintf("%d", c.ID)},
			},
		},
	}

	cardWidth := 52
	b.WriteString(ui.RenderCard(header, sections, cardWidth))
	b.WriteString("\n")

	// Actions with icons
	b.WriteString(ui.CardSectionStyle.Render("⚡ Actions") + "\n")
	actions := []struct {
		icon string
		text string
	}{
		{"✎", "Edit"},
		{"⚙", "Generate"},
		{"⎙", "Print"},
		{"✓", "Sign"},
		{"←", "Back"},
	}
	for i, action := range actions {
		cursor := "  "
		style := ui.MenuItemStyle
		if m.cursor == i {
			cursor = ui.CursorStyle.Render("▸ ")
			style = ui.SelectedMenuItemStyle
		}
		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, action.icon, style.Render(action.text)))
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

	// Card header with job ID and status
	header := ui.RenderCardHeader("◆", fmt.Sprintf("Print Job #%d %s", j.ID, ui.FormatStatus(j.Status)))

	completedAt := "In Progress"
	if j.CompletedAt != nil {
		completedAt = j.CompletedAt.Format(fmtDateTimeDisplay)
	}

	// Build sections
	sections := []ui.CardSection{
		{
			Title: "Job Info",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Contract ID", Value: fmt.Sprintf("%d", j.ContractID)},
				{Label: "Format", Value: j.Format},
				{Label: "Requested By", Value: j.RequestedBy},
			},
		},
		{
			Title: "Output",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "File Size", Value: formatFileSize(j.FileSize)},
				{Label: "Page Count", Value: fmt.Sprintf("%d pages", j.PageCount)},
			},
		},
		{
			Title: "Timeline",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "Queued At", Value: j.QueuedAt.Format(fmtDateTimeDisplay)},
				{Label: "Completed At", Value: completedAt},
			},
		},
	}

	cardWidth := 52
	b.WriteString(ui.RenderCard(header, sections, cardWidth))
	b.WriteString("\n")

	b.WriteString(ui.InfoStyle.Render("Press Esc to go back"))
	return b.String()
}

// formatFileSize formats bytes into human-readable size
func formatFileSize(bytes int64) string {
	// Handle negative values
	if bytes < 0 {
		return "-" + formatFileSize(-bytes)
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (m Model) renderSettings() string {
	var b strings.Builder

	// Card header
	header := ui.RenderCardHeader("◆", "Settings")

	// Build sections
	sections := []ui.CardSection{
		{
			Title: "Connection",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "API URL", Value: m.baseURL},
				{Label: "Token", Value: maskToken(m.token)},
			},
		},
		{
			Title: "Session",
			Icon:  "◈",
			Fields: []ui.CardField{
				{Label: "User", Value: m.user},
				{Label: "Tenant ID", Value: m.tenantID},
				{Label: "Signer", Value: m.signer},
			},
		},
	}

	cardWidth := 52
	b.WriteString(ui.RenderCard(header, sections, cardWidth))
	b.WriteString("\n")

	b.WriteString(ui.InfoStyle.Render("Set GPRINT_API_URL and GPRINT_TOKEN environment variables"))
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
