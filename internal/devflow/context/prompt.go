// Package context composes invoke prompts from task context documents.
package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// ComposeTaskPrompt assembles Tier-2 context for a run by fetching all
// attached task context documents and concatenating them.
// Returns empty string if no documents are attached.
func ComposeTaskPrompt(
	ctx context.Context,
	runID uuid.UUID,
	refStore store.TaskContextRefStore,
	docStore store.TaskContextStore,
) (string, error) {
	refs, err := refStore.ListByRun(ctx, runID)
	if err != nil {
		return "", fmt.Errorf("list context refs: %w", err)
	}
	if len(refs) == 0 {
		return "", nil
	}

	var b strings.Builder
	for _, ref := range refs {
		doc, err := docStore.Get(ctx, ref.TaskContextID)
		if err != nil {
			continue // skip missing documents
		}
		fmt.Fprintf(&b, "--- %s ---\n", doc.Title)
		b.WriteString(strings.TrimSpace(doc.Content))
		b.WriteString("\n\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n", nil
}
