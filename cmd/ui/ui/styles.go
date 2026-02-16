package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Message type constants for consistent UI messaging
const (
	// MessageTypeError indicates an error message style.
	MessageTypeError = "error"
	// MessageTypeSuccess indicates a success message style.
	MessageTypeSuccess = "success"
	// MessageTypeInfo indicates an informational message style.
	MessageTypeInfo = "info"
)

// DefaultCurrency is the fallback currency code when none is configured.
const DefaultCurrency = "BRL"

// ╔═══════════════════════════════════════════════════════════════════════════╗
// ║  NEON TERMINAL - Cyberpunk SSH Theme                                       ║
// ║  "Electric dreams in the digital void"                                     ║
// ╚═══════════════════════════════════════════════════════════════════════════╝
var (
	// ═══════════════════════════════════════════════════════════════════════════
	// CORE NEON PALETTE
	// ═══════════════════════════════════════════════════════════════════════════

	// Deep Dark Backgrounds
	bgVoid      = lipgloss.Color("#0a0a0f") // deepest background
	bgPrimary   = lipgloss.Color("#0d1117") // main workspace (shadow navy)
	bgSecondary = lipgloss.Color("#161b22") // panels, sidebars (dark slate)
	bgElevated  = lipgloss.Color("#1f2937") // modals, dropdowns (graphite)
	bgSubtle    = lipgloss.Color("#0a0a0f") // deep backgrounds (void black)
	bgSteel     = lipgloss.Color("#2d3748") // cards, elevated surfaces

	// Neon Accent Colors
	neonCyan    = lipgloss.Color("#00ffff") // Electric Cyan - primary neon
	neonMagenta = lipgloss.Color("#ff00ff") // Hot Magenta - secondary neon
	neonGreen   = lipgloss.Color("#39ff14") // Acid Green - success, active
	neonPurple  = lipgloss.Color("#bc13fe") // Plasma Purple - highlights
	neonOrange  = lipgloss.Color("#ff6600") // Nuclear Orange - warnings
	neonRed     = lipgloss.Color("#ff0055") // Laser Red - errors, critical
	neonPink    = lipgloss.Color("#ff10f0") // Neon Pink - special highlights
	neonBlue    = lipgloss.Color("#00d4ff") // Arctic Blue - info, links
	neonYellow  = lipgloss.Color("#ffff00") // Volt Yellow - alerts
	neonToxic   = lipgloss.Color("#00ff41") // Toxic Green - positive status

	// Foreground/Text Colors
	textPrimary   = lipgloss.Color("#f0f6fc") // ghost white - bright text
	textSecondary = lipgloss.Color("#c9d1d9") // silver - secondary text
	textMuted     = lipgloss.Color("#8b949e") // steel gray - muted text
	textBright    = lipgloss.Color("#ffffff") // pure white - primary text
	textDim       = lipgloss.Color("#4d5566") // dim gray - disabled

	// Border Colors
	borderDefault = lipgloss.Color("#30363d") // subtle border
	borderSubtle  = lipgloss.Color("#21262d") // very subtle border
	borderAccent  = neonCyan                  // neon cyan accent border
	borderGlow    = neonMagenta               // magenta glow border

	// Interactive States
	hoverBg    = lipgloss.Color("#21262d") // subtle hover
	selectedBg = lipgloss.Color("#1f2937") // selected background with cyan tint

	// Legacy color aliases (for compatibility)
	accentPrimary      = neonCyan
	accentPrimaryHover = neonBlue
	accentSuccess      = neonGreen
	accentWarning      = neonOrange
	accentError        = neonRed
	accentInfo         = neonBlue
	accentMagenta      = neonMagenta
	primaryColor       = neonCyan
	secondaryColor     = neonGreen
	warningColor       = neonOrange
	dangerColor        = neonRed
	mutedColor         = textMuted
	textColor          = textPrimary

	// ═══════════════════════════════════════════════════════════════════════════
	// LAYOUT DIMENSIONS
	// ═══════════════════════════════════════════════════════════════════════════

	// SidebarWidth is the expanded sidebar width in character cells.
	SidebarWidth = 28
	// SidebarCollapsedW is the collapsed sidebar width in character cells.
	SidebarCollapsedW = 4
	// HeaderHeight is the header height in terminal rows/lines.
	HeaderHeight = 3
	// FooterHeight is the footer height in terminal rows/lines.
	FooterHeight = 2

	// ═══════════════════════════════════════════════════════════════════════════
	// HEADER STYLES (Top Navigation / Menu Bar) - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	HeaderStyle = lipgloss.NewStyle().
			Background(bgSecondary).
			Foreground(textPrimary).
			Bold(true).
			Padding(0, 2).
			Height(HeaderHeight).
			BorderBottom(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderBottomForeground(neonCyan)

	HeaderTitleStyle = lipgloss.NewStyle().
				Foreground(neonCyan).
				Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(textMuted)

	BreadcrumbSeparatorStyle = lipgloss.NewStyle().
					Foreground(neonMagenta).
					Bold(true)

	BreadcrumbActiveStyle = lipgloss.NewStyle().
				Foreground(neonCyan).
				Bold(true)

	BreadcrumbIconStyle = lipgloss.NewStyle().
				Foreground(neonCyan)

	// ═══════════════════════════════════════════════════════════════════════════
	// SIDEBAR STYLES (Left Navigation) - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	SidebarStyle = lipgloss.NewStyle().
			Background(bgVoid).
			Padding(1, 0).
			BorderRight(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderRightForeground(neonMagenta)

	SidebarHeaderStyle = lipgloss.NewStyle().
				Foreground(neonMagenta).
				Bold(true).
				Padding(0, 2).
				MarginBottom(1)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Padding(0, 2)

	SidebarItemHoverStyle = lipgloss.NewStyle().
				Background(bgSteel).
				Foreground(neonCyan).
				Padding(0, 2)

	SidebarItemSelectedStyle = lipgloss.NewStyle().
					Background(bgElevated).
					Foreground(neonCyan).
					Bold(true).
					Padding(0, 2)

	SidebarToggleStyle = lipgloss.NewStyle().
				Foreground(neonMagenta).
				Padding(0, 1)

	// ═══════════════════════════════════════════════════════════════════════════
	// FOOTER STYLES (Bottom Bar) - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	FooterStyle = lipgloss.NewStyle().
			Background(bgVoid).
			Foreground(textSecondary).
			Padding(0, 2).
			Height(FooterHeight).
			BorderTop(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderTopForeground(neonCyan)

	FooterKeyStyle = lipgloss.NewStyle().
			Foreground(neonGreen).
			Bold(true)

	FooterLabelStyle = lipgloss.NewStyle().
				Foreground(textBright)

	FooterHelpStyle = lipgloss.NewStyle().
			Foreground(neonCyan)

	FooterStatusStyle = lipgloss.NewStyle().
				Foreground(neonGreen).
				Bold(true)

	FooterStatusWarningStyle = lipgloss.NewStyle().
					Foreground(neonOrange).
					Bold(true)

	FooterStatusErrorStyle = lipgloss.NewStyle().
				Foreground(neonRed).
				Bold(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// CONTENT AREA STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	ContentStyle = lipgloss.NewStyle().
			Background(bgPrimary).
			Padding(1, 2)

	// ═══════════════════════════════════════════════════════════════════════════
	// BASE STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(neonCyan).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(neonMagenta).
			MarginBottom(1)

	// ═══════════════════════════════════════════════════════════════════════════
	// MENU STYLES (Contextual Menu / Dropdown) - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(textSecondary)

	SelectedMenuItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(neonCyan).
				Bold(true)

	MenuHoverStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Background(bgElevated).
			Foreground(neonMagenta)

	MenuDisabledStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(textDim)

	CursorStyle = lipgloss.NewStyle().
			Foreground(neonMagenta).
			Bold(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// STATUS STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	StatusActiveStyle = lipgloss.NewStyle().
				Foreground(neonGreen).
				Bold(true)

	StatusInactiveStyle = lipgloss.NewStyle().
				Foreground(neonRed).
				Bold(true)

	StatusPendingStyle = lipgloss.NewStyle().
				Foreground(neonOrange).
				Bold(true)

	StatusInfoStyle = lipgloss.NewStyle().
			Foreground(neonBlue).
			Bold(true)

	StatusOfflineStyle = lipgloss.NewStyle().
				Foreground(textDim)

	// ═══════════════════════════════════════════════════════════════════════════
	// TABLE STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(neonCyan).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(neonMagenta)

	TableRowStyle = lipgloss.NewStyle().
			PaddingRight(2).
			Foreground(textSecondary)

	TableSelectedRowStyle = lipgloss.NewStyle().
				PaddingRight(2).
				Background(bgElevated).
				Foreground(neonCyan).
				Bold(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// FORM STYLES (Input Fields) - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	LabelStyle = lipgloss.NewStyle().
			Foreground(neonMagenta)

	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(borderDefault).
			Foreground(textPrimary).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.DoubleBorder()).
				BorderForeground(neonCyan).
				Foreground(textBright).
				Padding(0, 1)

	PlaceholderStyle = lipgloss.NewStyle().
				Foreground(textDim).
				Italic(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// MESSAGE STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	ErrorStyle = lipgloss.NewStyle().
			Foreground(neonRed).
			Bold(true).
			MarginTop(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(neonGreen).
			Bold(true).
			MarginTop(1)

	WarningStyle = lipgloss.NewStyle().
			Foreground(neonOrange).
			Bold(true).
			MarginTop(1)

	InfoStyle = lipgloss.NewStyle().
			Foreground(neonBlue).
			MarginTop(1)

	// ═══════════════════════════════════════════════════════════════════════════
	// HELP STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	HelpStyle = lipgloss.NewStyle().
			Foreground(textMuted).
			MarginTop(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(neonGreen).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(textSecondary)

	// ═══════════════════════════════════════════════════════════════════════════
	// BOX / DIALOG STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(neonCyan).
			Padding(1, 2)

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(neonMagenta).
			Background(bgElevated).
			Padding(1, 2)

	DialogTitleStyle = lipgloss.NewStyle().
				Background(bgSteel).
				Foreground(neonCyan).
				Bold(true).
				Padding(0, 1)

	// ═══════════════════════════════════════════════════════════════════════════
	// DETAIL VIEW STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	DetailKeyStyle = lipgloss.NewStyle().
			Foreground(neonMagenta).
			Width(20)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(textPrimary)

	// Card-based detail view styles
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(neonMagenta).
			Padding(1, 2).
			MarginBottom(1)

	CardHeaderStyle = lipgloss.NewStyle().
			Foreground(neonCyan).
			Bold(true).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderSubtle).
			MarginBottom(1).
			PaddingBottom(1)

	CardSectionStyle = lipgloss.NewStyle().
				Foreground(neonMagenta).
				Bold(true).
				MarginTop(1).
				MarginBottom(1)

	CardFieldLabelStyle = lipgloss.NewStyle().
				Foreground(textMuted).
				Width(16)

	CardFieldValueStyle = lipgloss.NewStyle().
				Foreground(textBright)

	CardFieldRowStyle = lipgloss.NewStyle().
				MarginBottom(0)

	CardDividerStyle = lipgloss.NewStyle().
				Foreground(borderSubtle)

	// Grid layout for 2-column display
	CardGridLeftStyle = lipgloss.NewStyle().
				Width(24).
				PaddingRight(2)

	CardGridRightStyle = lipgloss.NewStyle().
				Width(24)

	// ═══════════════════════════════════════════════════════════════════════════
	// BADGE STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	BadgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Background(neonCyan).
			Foreground(bgVoid).
			Bold(true)

	BadgeSuccessStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(neonGreen).
				Foreground(bgVoid).
				Bold(true)

	BadgeDangerStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(neonRed).
				Foreground(textBright).
				Bold(true)

	BadgeWarningStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Background(neonOrange).
				Foreground(bgVoid).
				Bold(true)

	BadgeInfoStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Background(neonBlue).
			Foreground(bgVoid).
			Bold(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// BUTTON STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	ButtonPrimaryStyle = lipgloss.NewStyle().
				Background(neonCyan).
				Foreground(bgVoid).
				Padding(0, 2).
				Bold(true)

	ButtonPrimaryHoverStyle = lipgloss.NewStyle().
				Background(textBright).
				Foreground(bgVoid).
				Padding(0, 2).
				Bold(true)

	ButtonSecondaryStyle = lipgloss.NewStyle().
				Background(bgElevated).
				Foreground(neonMagenta).
				Padding(0, 2).
				Bold(true)

	ButtonSecondaryHoverStyle = lipgloss.NewStyle().
					Background(bgSteel).
					Foreground(neonMagenta).
					Padding(0, 2).
					Bold(true)

	ButtonDangerStyle = lipgloss.NewStyle().
				Background(neonRed).
				Foreground(textBright).
				Padding(0, 2).
				Bold(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// PROGRESS / LOADING STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	ProgressBarStyle = lipgloss.NewStyle().
				Background(bgSteel)

	ProgressFillStyle = lipgloss.NewStyle().
				Background(neonCyan)

	ProgressTextStyle = lipgloss.NewStyle().
				Foreground(neonCyan).
				Bold(true)

	// ═══════════════════════════════════════════════════════════════════════════
	// NOTIFICATION TOAST STYLES - Neon Edition
	// ═══════════════════════════════════════════════════════════════════════════

	ToastStyle = lipgloss.NewStyle().
			Background(bgElevated).
			Border(lipgloss.DoubleBorder()).
			BorderForeground(neonCyan).
			Padding(0, 2)

	ToastSuccessIconStyle = lipgloss.NewStyle().
				Foreground(neonGreen).
				Bold(true)

	ToastWarningIconStyle = lipgloss.NewStyle().
				Foreground(neonOrange).
				Bold(true)

	ToastErrorIconStyle = lipgloss.NewStyle().
				Foreground(neonRed).
				Bold(true)

	ToastInfoIconStyle = lipgloss.NewStyle().
				Foreground(neonBlue).
				Bold(true)
)

// FormatStatus returns a styled status string for domain statuses.
// It normalizes the input to uppercase for case-insensitive matching.
// For boolean strings, use FormatBool instead.
func FormatStatus(status string) string {
	normalized := strings.ToUpper(status)
	switch normalized {
	case "ACTIVE", "COMPLETED", "SUCCESS", "ONLINE":
		return StatusActiveStyle.Render(normalized)
	case "DRAFT", "PENDING", "QUEUED", "PROCESSING", "BUSY":
		return StatusPendingStyle.Render(normalized)
	case "CANCELLED", "FAILED", "SUSPENDED", "ERROR":
		return StatusInactiveStyle.Render(normalized)
	case "OFFLINE", "DISABLED", "INACTIVE":
		return StatusOfflineStyle.Render(normalized)
	case "INFO", "NOTE":
		return StatusInfoStyle.Render(normalized)
	default:
		return StatusInfoStyle.Render(normalized)
	}
}

// FormatBool returns a styled boolean
func FormatBool(b bool) string {
	if b {
		return StatusActiveStyle.Render("Yes")
	}
	return StatusInactiveStyle.Render("No")
}

// FormatKey renders a keyboard shortcut key in the theme style
func FormatKey(key string) string {
	return FooterKeyStyle.Render(key)
}

// FormatHelpItem renders a help item as "Key Label"
func FormatHelpItem(key, label string) string {
	return FooterKeyStyle.Render(key) + " " + FooterLabelStyle.Render(label)
}

// FormatBreadcrumb renders breadcrumb segments with neon separators
func FormatBreadcrumb(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	if len(segments) == 1 {
		return BreadcrumbActiveStyle.Render(segments[0])
	}

	var result strings.Builder
	for i, seg := range segments {
		if i == len(segments)-1 {
			result.WriteString(BreadcrumbActiveStyle.Render(seg))
		} else {
			result.WriteString(BreadcrumbStyle.Render(seg))
			result.WriteString(BreadcrumbSeparatorStyle.Render(" ▸ "))
		}
	}
	return result.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// CARD RENDERING HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// CardField represents a field in a detail card
type CardField struct {
	Label string
	Value string
	Icon  string // optional icon prefix
}

// CardSection represents a section with multiple fields
type CardSection struct {
	Title  string
	Icon   string
	Fields []CardField
}

// RenderCardHeader renders a card header with icon and title
func RenderCardHeader(icon, title string) string {
	return CardHeaderStyle.Render(icon + " " + title)
}

// RenderCardField renders a single field row
func RenderCardField(f CardField) string {
	label := CardFieldLabelStyle.Render(f.Label)
	value := CardFieldValueStyle.Render(f.Value)
	if f.Icon != "" {
		return f.Icon + " " + label + value
	}
	return "  " + label + value
}

// RenderCardSection renders a section with title and fields
func RenderCardSection(s CardSection) string {
	var b strings.Builder
	if s.Title != "" {
		title := s.Title
		if s.Icon != "" {
			title = s.Icon + " " + title
		}
		b.WriteString(CardSectionStyle.Render(title) + "\n")
	}
	for _, f := range s.Fields {
		b.WriteString(RenderCardField(f) + "\n")
	}
	return b.String()
}

// RenderCardDivider renders a horizontal divider
func RenderCardDivider(width int) string {
	return CardDividerStyle.Render(strings.Repeat("─", width))
}

// RenderCard renders a complete card with sections
func RenderCard(header string, sections []CardSection, width int) string {
	var b strings.Builder
	b.WriteString(header + "\n\n")
	for i, s := range sections {
		b.WriteString(RenderCardSection(s))
		if i < len(sections)-1 {
			b.WriteString(RenderCardDivider(width-6) + "\n")
		}
	}
	return CardStyle.Width(width).Render(b.String())
}

// RenderTwoColumnCard renders fields in a 2-column layout
func RenderTwoColumnCard(header string, leftFields, rightFields []CardField, width int) string {
	var b strings.Builder
	b.WriteString(header + "\n\n")

	maxRows := len(leftFields)
	if len(rightFields) > maxRows {
		maxRows = len(rightFields)
	}

	for i := 0; i < maxRows; i++ {
		left := ""
		right := ""
		if i < len(leftFields) {
			left = RenderCardField(leftFields[i])
		}
		if i < len(rightFields) {
			right = RenderCardField(rightFields[i])
		}
		leftCol := CardGridLeftStyle.Render(left)
		rightCol := CardGridRightStyle.Render(right)
		b.WriteString(leftCol + rightCol + "\n")
	}

	return CardStyle.Width(width).Render(b.String())
}
