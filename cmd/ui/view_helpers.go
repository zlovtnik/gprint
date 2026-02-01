package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/zlovtnik/gprint/cmd/ui/ui"
)

// ═══════════════════════════════════════════════════════════════════════════
// DETAIL VIEW HELPERS - Render key-value detail views
// ═══════════════════════════════════════════════════════════════════════════

// DetailItem represents a key-value pair for display.
type DetailItem struct {
	Key   string
	Value string
}

// RenderDetailView renders a standard detail view with title, details, and actions.
func RenderDetailView(title string, details []DetailItem, actions []string, cursor int) string {
	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render(title) + "\n\n")

	// Render detail items
	for _, d := range details {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.Key+":"),
			ui.DetailValueStyle.Render(d.Value)))
	}

	// Render actions
	if len(actions) > 0 {
		b.WriteString("\n" + ui.SubtitleStyle.Render("Actions") + "\n")
		for i, action := range actions {
			cursorSymbol, style := renderCursor(cursor == i)
			b.WriteString(fmt.Sprintf(fmtCursorItem, cursorSymbol, style.Render(action)))
		}
	}

	return b.String()
}

// RenderKeyValueList renders a list of key-value pairs.
func RenderKeyValueList(items []DetailItem) string {
	var b strings.Builder
	for _, d := range items {
		b.WriteString(fmt.Sprintf(fmtKeyValue,
			ui.DetailKeyStyle.Render(d.Key+":"),
			ui.DetailValueStyle.Render(d.Value)))
	}
	return b.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// FORM HELPERS - Render form views with labels and inputs
// ═══════════════════════════════════════════════════════════════════════════

// FormConfig holds form rendering configuration.
type FormConfig struct {
	Title     string
	EditTitle string // Title when editing (if different)
	Labels    []string
	IsEdit    bool
	HelpText  string
}

// RenderForm renders a form view with labels and inputs.
// Returns an error message if input count doesn't match label count.
func RenderForm(cfg FormConfig, inputs []interface{ View() string }) string {
	var b strings.Builder

	// Determine title
	title := cfg.Title
	if cfg.IsEdit && cfg.EditTitle != "" {
		title = cfg.EditTitle
	}
	b.WriteString(ui.SubtitleStyle.Render(title) + "\n\n")

	// Warn if inputs and labels mismatch
	if len(inputs) != len(cfg.Labels) {
		b.WriteString(ui.ErrorStyle.Render(
			fmt.Sprintf("[Form Error: expected %d inputs, got %d]\n\n", len(cfg.Labels), len(inputs))))
	}

	// Use bounded iteration to prevent index out of range
	maxItems := len(cfg.Labels)
	if len(inputs) < maxItems {
		maxItems = len(inputs)
	}

	for i := 0; i < maxItems; i++ {
		label := ui.LabelStyle.Render(cfg.Labels[i] + ":")
		b.WriteString(fmt.Sprintf(fmtLabelInput, label, inputs[i].View()))
	}

	// Help text
	helpText := cfg.HelpText
	if helpText == "" {
		helpText = msgFormSaveCancel
	}
	b.WriteString(ui.InfoStyle.Render(helpText))

	return b.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// ACTION MENU HELPERS - Render action menus with cursor
// ═══════════════════════════════════════════════════════════════════════════

// RenderActionMenu renders a vertical menu of actions with cursor support.
func RenderActionMenu(actions []string, cursor int) string {
	var b strings.Builder
	for i, action := range actions {
		c, style := renderCursor(cursor == i)
		b.WriteString(fmt.Sprintf(fmtCursorItem, c, style.Render(action)))
	}
	return b.String()
}

// RenderActionMenuWithTitle renders an action menu with a section title.
func RenderActionMenuWithTitle(title string, actions []string, cursor int) string {
	var b strings.Builder
	b.WriteString("\n" + ui.SubtitleStyle.Render(title) + "\n")
	b.WriteString(RenderActionMenu(actions, cursor))
	return b.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// EMPTY STATE HELPERS - Render empty/no-data states
// ═══════════════════════════════════════════════════════════════════════════

// RenderEmptyState renders an empty state view with a message and back option.
func RenderEmptyState(title, message string, cursor int) string {
	var b strings.Builder
	b.WriteString(ui.SubtitleStyle.Render(title) + "\n\n")
	b.WriteString(ui.InfoStyle.Render(message) + "\n\n")

	c, style := renderCursor(cursor == 0)
	b.WriteString(fmt.Sprintf(fmtMenuItemNL, c, style.Render(backToMainMenu)))
	return b.String()
}

// ═══════════════════════════════════════════════════════════════════════════
// FORMAT HELPERS - Common formatting utilities
// ═══════════════════════════════════════════════════════════════════════════

// FormatOptionalTime formats a *time.Time, returning fallback if nil.
// Uses concrete *time.Time to correctly handle typed nil pointers.
func FormatOptionalTime(t *time.Time, format, fallback string) string {
	if t == nil {
		return fallback
	}
	return t.Format(format)
}

// FormatID formats an int64 ID for display.
func FormatID(id int64) string {
	return fmt.Sprintf("%d", id)
}

// FormatIDf formats an ID with a format string.
func FormatIDf(format string, id int64) string {
	return fmt.Sprintf(format, id)
}

// ═══════════════════════════════════════════════════════════════════════════
// ROW RENDERING HELPERS - Build list item rows
// ═══════════════════════════════════════════════════════════════════════════

// RowBuilder helps construct formatted list rows.
type RowBuilder struct {
	parts  []string
	cursor string
	style  lipgloss.Style
}

// NewRowBuilder creates a new RowBuilder with cursor state.
func NewRowBuilder(selected bool) *RowBuilder {
	cursor, style := renderCursor(selected)
	return &RowBuilder{
		cursor: cursor,
		style:  style,
	}
}

// Add adds a column to the row.
func (r *RowBuilder) Add(value string) *RowBuilder {
	r.parts = append(r.parts, r.style.Render(value))
	return r
}

// AddFixed adds a fixed-width column to the row.
func (r *RowBuilder) AddFixed(format string, value string) *RowBuilder {
	r.parts = append(r.parts, r.style.Render(fmt.Sprintf(format, value)))
	return r
}

// AddPlain adds a column without applying the style.
func (r *RowBuilder) AddPlain(value string) *RowBuilder {
	r.parts = append(r.parts, value)
	return r
}

// Build returns the complete row string.
func (r *RowBuilder) Build() string {
	return r.cursor + strings.Join(r.parts, " | ") + "\n"
}

// Note: backToMainMenu and fmtMenuItemNL are defined in main.go
