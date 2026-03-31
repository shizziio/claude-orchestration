// Package claudemd composes CLAUDE.md files from project context entries.
package claudemd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

const maxBudgetBytes = 4096

// Compose renders CLAUDE.md content from project context entries and team definitions.
// Entries are grouped by doc_type (rules, index, structure) and ordered by sort_order.
// teamStore may be nil (teams section will be skipped).
func Compose(ctx context.Context, projectID uuid.UUID, ctxStore store.ProjectContextStore, teamStore store.ProjectTeamStore) (string, error) {
	entries, err := ctxStore.ListByProject(ctx, projectID, "")
	if err != nil {
		return "", fmt.Errorf("list project context: %w", err)
	}

	var teams []*store.ProjectTeam
	if teamStore != nil {
		teams, _ = teamStore.ListByProject(ctx, projectID)
	}

	if len(entries) == 0 && len(teams) == 0 {
		return "", nil
	}

	var b strings.Builder

	// Group entries by doc_type in display order
	typeOrder := []struct {
		docType string
		heading string
	}{
		{"rules", "Project Rules"},
		{"index", "Document Index"},
		{"structure", "Project Structure"},
	}

	for _, t := range typeOrder {
		var sectionEntries []*store.ProjectContext
		for _, e := range entries {
			if e.DocType == t.docType {
				sectionEntries = append(sectionEntries, e)
			}
		}
		if len(sectionEntries) == 0 {
			continue
		}

		fmt.Fprintf(&b, "## %s\n\n", t.heading)
		for _, e := range sectionEntries {
			if e.Title != "" && e.Title != t.heading {
				fmt.Fprintf(&b, "### %s\n\n", e.Title)
			}
			b.WriteString(strings.TrimSpace(e.Content))
			b.WriteString("\n\n")
		}
	}

	// Include any entries with unknown doc_type
	for _, e := range entries {
		if e.DocType != "rules" && e.DocType != "index" && e.DocType != "structure" {
			fmt.Fprintf(&b, "## %s\n\n", e.Title)
			b.WriteString(strings.TrimSpace(e.Content))
			b.WriteString("\n\n")
		}
	}

	// Teams section
	if len(teams) > 0 {
		b.WriteString("## Teams\n\n")
		for _, t := range teams {
			fmt.Fprintf(&b, "### %s\n\n", t.TeamName)
			if t.Description != nil && *t.Description != "" {
				b.WriteString(*t.Description)
				b.WriteString("\n\n")
			}
			if len(t.TeamConfig) > 2 { // skip empty "{}"
				fmt.Fprintf(&b, "```json\n%s\n```\n\n", string(t.TeamConfig))
			}
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// WriteToWorkspace writes the CLAUDE.md content to the project workspace.
// Creates or overwrites {workspacePath}/CLAUDE.md.
func WriteToWorkspace(content, workspacePath string) error {
	if workspacePath == "" {
		return fmt.Errorf("workspace path is empty")
	}
	if content == "" {
		return nil // nothing to write
	}
	target := filepath.Join(workspacePath, "CLAUDE.md")
	return os.WriteFile(target, []byte(content), 0644)
}
