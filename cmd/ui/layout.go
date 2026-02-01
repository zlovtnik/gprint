package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

// renderLayout renders the full application layout with header, sidebar, content, and footer
func (m Model) renderLayout() string {
	// Calculate dimensions
	sidebarWidth := 0
	if m.sidebarOpen {
		sidebarWidth = ui.SidebarWidth
	} else {
		sidebarWidth = ui.SidebarCollapsedW
	}

	contentWidth := m.width - sidebarWidth
	if contentWidth < 20 {
		contentWidth = 20
	}

	contentHeight := m.height - ui.HeaderHeight - ui.FooterHeight
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Render components
	header := m.renderHeader(m.width)
	sidebar := m.renderSidebar(sidebarWidth, contentHeight)
	content := m.renderContent(contentWidth, contentHeight)
	footer := m.renderFooter(m.width)

	// Combine sidebar and content horizontally
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	// Combine header, main area, and footer vertically
	return lipgloss.JoinVertical(lipgloss.Left, header, mainArea, footer)
}

// renderHeader renders the fixed top header with breadcrumb
func (m Model) renderHeader(width int) string {
	// Neon Logo and title
	logo := ui.HeaderTitleStyle.Render("⚡ GPrint")

	// Breadcrumb with neon styled separator
	breadcrumb := m.getBreadcrumb()
	var crumbParts []string
	for i, crumb := range breadcrumb {
		if i == len(breadcrumb)-1 {
			crumbParts = append(crumbParts, ui.BreadcrumbActiveStyle.Render(crumb))
		} else {
			crumbParts = append(crumbParts, ui.BreadcrumbStyle.Render(crumb))
		}
	}
	sep := ui.BreadcrumbSeparatorStyle.Render(" ▸ ")
	crumbStr := strings.Join(crumbParts, sep)

	// Status indicator with neon styling
	status := ""
	if m.token != "" {
		userInfo := ""
		if m.user != "" {
			userInfo = m.user + " "
		}
		status = ui.StatusActiveStyle.Render("◉") + " " + ui.FooterLabelStyle.Render(userInfo+"Online")
	} else {
		status = ui.StatusOfflineStyle.Render("○ Offline")
	}

	// Layout: logo | breadcrumb | status
	leftPart := logo + "  " + crumbStr
	rightPart := status

	// Calculate spacing: subtract 4 to account for HeaderStyle's horizontal padding (0, 2) which adds 4 total chars
	spacing := width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - 4
	if spacing < 1 {
		spacing = 1
	}

	headerContent := leftPart + strings.Repeat(" ", spacing) + rightPart

	return ui.HeaderStyle.Width(width).Render(headerContent)
}

// sidebarItemState determines the visual state of a sidebar item
type sidebarItemState struct {
	style  lipgloss.Style
	cursor string
}

// getSidebarItemState returns the style and cursor for a sidebar item
func (m Model) getSidebarItemState(index int, isCurrentView bool) sidebarItemState {
	isHovered := m.focusOnSidebar && m.sidebarCursor == index

	switch {
	case isHovered:
		return sidebarItemState{ui.SidebarItemSelectedStyle, "▸ "}
	case isCurrentView:
		return sidebarItemState{ui.SidebarItemHoverStyle, "▹ "}
	default:
		return sidebarItemState{ui.SidebarItemStyle, "  "}
	}
}

// renderSidebarOpen renders the expanded sidebar
func (m Model) renderSidebarOpen(items []SidebarItem, width int) string {
	var b strings.Builder

	toggleHint := ui.SidebarToggleStyle.Render("[Ctrl+B]")
	b.WriteString(ui.SidebarHeaderStyle.Render("▓ MENU ▓") + " " + toggleHint + "\n\n")

	currentIdx := m.getSidebarIndexForView()
	for i, item := range items {
		state := m.getSidebarItemState(i, currentIdx == i)
		line := fmt.Sprintf("%s%s %s", state.cursor, item.Icon, item.Title)
		b.WriteString(state.style.Width(width-2).Render(line) + "\n")
	}
	return b.String()
}

// renderSidebarCollapsed renders the collapsed sidebar (icons only)
func (m Model) renderSidebarCollapsed(items []SidebarItem) string {
	var b strings.Builder
	b.WriteString(ui.SidebarToggleStyle.Render("▓\n\n"))

	currentIdx := m.getSidebarIndexForView()
	for i, item := range items {
		state := m.getSidebarItemState(i, currentIdx == i)
		b.WriteString(state.style.Render(item.Icon) + "\n")
	}
	return b.String()
}

// renderSidebar renders the collapsible sidebar menu
func (m Model) renderSidebar(width, height int) string {
	items := getSidebarItems()

	var content string
	if m.sidebarOpen {
		content = m.renderSidebarOpen(items, width)
	} else {
		content = m.renderSidebarCollapsed(items)
	}

	// Fill remaining height
	lines := strings.Count(content, "\n")
	padding := strings.Repeat("\n", max(0, height-1-lines))

	return ui.SidebarStyle.Width(width).Height(height).Render(content + padding)
}

// renderContent renders the main content area
func (m Model) renderContent(width, height int) string {
	var content string

	switch m.view {
	case ui.ViewMain:
		content = m.renderDashboard()
	case ui.ViewCustomers:
		content = m.renderCustomerList()
	case ui.ViewCustomerDetail:
		content = m.renderCustomerDetail()
	case ui.ViewCustomerCreate, ui.ViewCustomerEdit:
		content = m.renderCustomerForm()
	case ui.ViewServices:
		content = m.renderServiceList()
	case ui.ViewServiceDetail:
		content = m.renderServiceDetail()
	case ui.ViewServiceCreate, ui.ViewServiceEdit:
		content = m.renderServiceForm()
	case ui.ViewContracts:
		content = m.renderContractList()
	case ui.ViewContractDetail:
		content = m.renderContractDetail()
	case ui.ViewContractCreate, ui.ViewContractEdit:
		content = m.renderContractForm()
	case ui.ViewPrintJobs:
		content = m.renderPrintJobList()
	case ui.ViewPrintJobDetail:
		content = m.renderPrintJobDetail()
	case ui.ViewSettings:
		content = m.renderSettings()
	default:
		content = m.renderDashboard()
	}

	// Add message if present
	if m.message != "" {
		var msgStyle lipgloss.Style
		switch m.messageType {
		case ui.MessageTypeError:
			msgStyle = ui.ErrorStyle
		case ui.MessageTypeSuccess:
			msgStyle = ui.SuccessStyle
		default:
			msgStyle = ui.InfoStyle
		}
		content += "\n" + msgStyle.Render(m.message)
	}

	return ui.ContentStyle.Width(width).Height(height).Render(content)
}

// renderFooter renders the fixed bottom footer with help and status
func (m Model) renderFooter(width int) string {
	// Help text based on context (already styled)
	help := m.getContextualHelp()

	// Right side: API info with status icon
	statusIcon := ui.StatusActiveStyle.Render("●")
	if m.token == "" {
		statusIcon = ui.StatusOfflineStyle.Render("○")
	}
	apiInfo := statusIcon + " " + ui.FooterLabelStyle.Render(m.baseURL)

	// Calculate spacing
	spacing := width - lipgloss.Width(help) - lipgloss.Width(apiInfo) - 4
	if spacing < 1 {
		spacing = 1
	}

	footerContent := help + strings.Repeat(" ", spacing) + apiInfo

	return ui.FooterStyle.Width(width).Render(footerContent)
}

// getContextualHelp returns styled help text based on current context
func (m Model) getContextualHelp() string {
	// Helper for styled shortcut - neon style
	key := func(k string) string { return ui.FooterKeyStyle.Render(k) }
	lbl := func(l string) string { return ui.FooterLabelStyle.Render(l) }
	sep := ui.FooterHelpStyle.Render(" ║ ")

	base := key("Ctrl+B") + " " + lbl("Menu")

	if m.focusOnSidebar {
		return base + sep + key("↑↓") + " " + lbl("Nav") + sep + key("Enter") + " " + lbl("Select") + sep + key("→") + " " + lbl("Content")
	}

	switch m.view {
	case ui.ViewMain:
		return base + sep + key("←") + " " + lbl("Menu") + sep + key("q") + " " + lbl("Quit")
	case ui.ViewCustomers, ui.ViewServices, ui.ViewContracts, ui.ViewPrintJobs:
		return base + sep + key("n") + " " + lbl("New") + sep + key("r") + " " + lbl("Refresh") + sep + key("Esc") + " " + lbl("Back")
	case ui.ViewCustomerDetail, ui.ViewServiceDetail, ui.ViewPrintJobDetail:
		return base + sep + key("e") + " " + lbl("Edit") + sep + key("d") + " " + lbl("Delete") + sep + key("Esc") + " " + lbl("Back")
	case ui.ViewContractDetail:
		return base + sep + key("e") + " " + lbl("Edit") + sep + key("Esc") + " " + lbl("Back")
	case ui.ViewSettings:
		return base + sep + key("Esc") + " " + lbl("Back")
	case ui.ViewCustomerCreate, ui.ViewCustomerEdit,
		ui.ViewServiceCreate, ui.ViewServiceEdit,
		ui.ViewContractCreate, ui.ViewContractEdit:
		return key("Tab") + " " + lbl("Next") + sep + key("⇧Tab") + " " + lbl("Prev") + sep + key("Enter") + " " + lbl("Save") + sep + key("Esc") + " " + lbl("Cancel")
	default:
		return base + sep + key("Esc") + " " + lbl("Back") + sep + key("q") + " " + lbl("Quit")
	}
}

// renderDashboard renders the main dashboard view
func (m Model) renderDashboard() string {
	var b strings.Builder

	// Cyberpunk welcome banner
	b.WriteString(ui.TitleStyle.Render("╔════════════════════════════════════╗") + "\n")
	b.WriteString(ui.TitleStyle.Render("║") + "  " + ui.HeaderTitleStyle.Render("⚡ SYSTEM ONLINE ⚡") + "  " + ui.TitleStyle.Render("║") + "\n")
	b.WriteString(ui.TitleStyle.Render("╚════════════════════════════════════╝") + "\n\n")

	// Quick stats with neon styling
	b.WriteString(ui.SubtitleStyle.Render("▓▓ SYSTEM STATUS ▓▓") + "\n\n")

	stats := []struct {
		icon  string
		label string
		value string
	}{
		{"◈", "Customers", fmt.Sprintf("%d", len(m.customers))},
		{"◇", "Services", fmt.Sprintf("%d", len(m.services))},
		{"◆", "Contracts", fmt.Sprintf("%d", len(m.contracts))},
		{"▣", labelPrintJobs, fmt.Sprintf("%d", len(m.printJobs))},
	}

	for _, stat := range stats {
		b.WriteString(fmt.Sprintf("  %s %s: %s\n",
			ui.CursorStyle.Render(stat.icon),
			ui.DetailKeyStyle.Render(stat.label),
			ui.StatusActiveStyle.Render(stat.value)))
	}

	b.WriteString("\n" + ui.InfoStyle.Render("◀ Use ← arrow or Ctrl+B to access the menu"))

	return b.String()
}
