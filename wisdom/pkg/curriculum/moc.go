package curriculum

import (
	"fmt"
	"strings"
)

// GenerateMOC creates a new Map of Content markdown string.
func GenerateMOC(title string, initialLinks []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s MOC\n\n", title))
	sb.WriteString("Welcome to the Map of Content. This serves as the root node for navigating this domain.\n\n")
	sb.WriteString("## Concepts\n")

	for _, link := range initialLinks {
		sb.WriteString(fmt.Sprintf("- [[%s]]\n", link))
	}

	return sb.String()
}

// AppendToMOC parses an existing MOC payload and deterministically injects a new wikilink.
func AppendToMOC(existingMOC string, newLink string) string {
	wikilink := fmt.Sprintf("[[%s]]", newLink)
	
	// Avoid duplicates
	if strings.Contains(existingMOC, wikilink) {
		return existingMOC
	}

	lines := strings.Split(existingMOC, "\n")
	
	var sb strings.Builder
	inserted := false

	for _, line := range lines {
		sb.WriteString(line + "\n")
		// Insert immediately after the ## Concepts header
		if strings.TrimSpace(line) == "## Concepts" && !inserted {
			sb.WriteString(fmt.Sprintf("- %s\n", wikilink))
			inserted = true
		}
	}

	// Fallback if the header wasn't found
	if !inserted {
		sb.WriteString(fmt.Sprintf("\n## Uncategorized\n- %s\n", wikilink))
	}

	return strings.TrimRight(sb.String(), "\n") // Clean up trailing newline
}
