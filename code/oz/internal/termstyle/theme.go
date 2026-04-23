// Package termstyle holds the oz CLI terminal color palette and shared lipgloss styles.
// All TTY theming should go through here so hex values and role→style mapping stay in one place.
package termstyle

import "github.com/charmbracelet/lipgloss"

// Brand colors for interactive terminal output (aligned with docs and web).
var (
	Purple = lipgloss.Color("#7C3AED")
	Faint  = lipgloss.Color("#6B7280")
	Green  = lipgloss.Color("#10B981")
	Lavend = lipgloss.Color("#A78BFA")
	Soft   = lipgloss.Color("#D1D5DB")
	Orange = lipgloss.Color("#F59E0B")
	Red    = lipgloss.Color("#EF4444")
	White  = lipgloss.Color("#FFFFFF")
)

// Shared lipgloss styles used across cmd, audit, review, and other TTY surfaces.
var (
	Brand    = lipgloss.NewStyle().Bold(true).Foreground(Purple)
	Subtle   = lipgloss.NewStyle().Foreground(Faint)
	OK       = lipgloss.NewStyle().Bold(true).Foreground(Green)
	Command  = lipgloss.NewStyle().Foreground(Lavend).Bold(true)
	TreeRoot = lipgloss.NewStyle().Bold(true).Foreground(Purple)
	TreeDir  = lipgloss.NewStyle().Foreground(Lavend)
	TreeFile = lipgloss.NewStyle().Foreground(Soft)
	Section  = lipgloss.NewStyle().Bold(true).Foreground(Soft)

	// Review / diff / secondary emphasis
	AccentBold = lipgloss.NewStyle().Bold(true).Foreground(Lavend)
	Muted      = lipgloss.NewStyle().Foreground(Soft)
	Warn       = lipgloss.NewStyle().Foreground(Orange)

	// Audit human report
	AuditTitle  = Brand
	AuditCode   = AccentBold
	AuditPath   = Subtle
	AuditOKLine = OK
	AuditRule   = Subtle
	AuditHint   = Subtle
	SectionErr  = lipgloss.NewStyle().Bold(true).Foreground(Red)
	SectionWarn = lipgloss.NewStyle().Bold(true).Foreground(Orange)
	SectionInfo = lipgloss.NewStyle().Bold(true).Foreground(Lavend)

	// Audit summary / footer counts
	CountError = lipgloss.NewStyle().Foreground(Red).Bold(true)
	CountWarn  = lipgloss.NewStyle().Foreground(Orange).Bold(true)
	CountInfo  = lipgloss.NewStyle().Foreground(Lavend)

	// ASCII banner (logo) glyph colors
	BannerSolid = lipgloss.NewStyle().Foreground(Purple)
	BannerShade = lipgloss.NewStyle().Foreground(Lavend)
)
