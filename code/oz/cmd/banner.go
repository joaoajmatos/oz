package cmd

import (
	"fmt"
	"strings"

	"github.com/joaoajmatos/oz/internal/termstyle"
)

// bannerLines is the oz ASCII art logo split by line so that solid (█) and
// shade (▒) characters can be coloured independently.
var bannerLines = []string{
	`    ███████    ███████████`,
	`  ███▒▒▒▒▒███ ▒█▒▒▒▒▒▒███ `,
	` ███     ▒▒███▒     ███▒  `,
	`▒███      ▒███     ███    `,
	`▒███      ▒███    ███     `,
	`▒▒███     ███   ████     █`,
	` ▒▒▒███████▒   ███████████`,
	`   ▒▒▒▒▒▒▒    ▒▒▒▒▒▒▒▒▒▒▒ `,
}

// renderBannerLine renders a single banner line colouring █ in deep purple
// and ▒ in lavender so the shadow effect reads clearly in terminal.
func renderBannerLine(line string) string {
	var sb strings.Builder
	for _, r := range line {
		switch r {
		case '█':
			sb.WriteString(termstyle.BannerSolid.Render(string(r)))
		case '▒':
			sb.WriteString(termstyle.BannerShade.Render(string(r)))
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// PrintBanner writes the oz logo banner to stdout.
func PrintBanner() {
	fmt.Println()
	for _, l := range bannerLines {
		fmt.Println(renderBannerLine(l))
	}
	fmt.Println()
}
