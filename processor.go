package main

import (
	"fmt"
	"log"
	"strings"

	miniflux "miniflux.app/v2/client"
)

// Processor handles the processing of entries against rules
type Processor struct {
	client  MinifluxClient
	matcher *Matcher
	logger  *log.Logger
	dryRun  bool
}

// NewProcessor creates a new Processor
func NewProcessor(
	client MinifluxClient,
	matcher *Matcher,
	logger *log.Logger,
	dryRun bool,
) *Processor {
	return &Processor{
		client:  client,
		matcher: matcher,
		logger:  logger,
		dryRun:  dryRun,
	}
}

// ProcessStats holds statistics about a processing run
type ProcessStats struct {
	TotalEntries   int
	MatchedEntries int
	MarkedRead     int
	Removed        int
	Replaced       int
	Errors         int
}

// Process fetches unread entries and applies matching rules
func (p *Processor) Process() (*ProcessStats, error) {
	stats := &ProcessStats{}

	// Fetch entries (unread by default, all in dry-run)
	filter := &miniflux.Filter{
		Limit: 100, // Process in batches
	}
	if !p.dryRun {
		filter.Status = miniflux.EntryStatusUnread
	}

	offset := 0
	for {
		filter.Offset = offset
		result, err := p.client.Entries(filter)
		if err != nil {
			return stats, fmt.Errorf("failed to fetch entries: %w", err)
		}

		if len(result.Entries) == 0 {
			break
		}

		for _, entry := range result.Entries {
			stats.TotalEntries++
			p.processEntry(entry, stats)
		}

		offset += len(result.Entries)

		// Check if we've processed all entries
		if offset >= result.Total {
			break
		}
	}

	return stats, nil
}

// processEntry processes a single entry against all rules
func (p *Processor) processEntry(entry *miniflux.Entry, stats *ProcessStats) {
	result := p.matcher.Match(entry)
	if !result.Matched {
		return
	}

	stats.MatchedEntries++

	feedTitle := ""
	if entry.Feed != nil {
		feedTitle = entry.Feed.Title
	}

	p.logger.Printf("Rule '%s' matched entry: [%s] %s", result.Rule.Name, feedTitle, entry.Title)

	var status string
	switch result.Action {
	case actionRead:
		status = miniflux.EntryStatusRead
		stats.MarkedRead++
	case actionRemove:
		status = miniflux.EntryStatusRemoved
		stats.Removed++
	case actionReplace:
		p.replaceEntryText(entry, feedTitle, result.Rule, stats)
		return
	default:
		p.logger.Printf("Unknown action '%s' for rule '%s'", result.Action, result.Rule.Name)
		stats.Errors++
		return
	}

	if p.dryRun {
		actionVerb := result.Action
		if result.Action == actionRead {
			actionVerb = "mark read"
		} else if result.Action == actionRemove {
			actionVerb = "remove"
		}
		p.logger.Printf(
			"Dry run: would %s entry %d [%s] %s",
			actionVerb,
			entry.ID,
			feedTitle,
			entry.Title,
		)
		return
	}

	if err := p.client.UpdateEntries([]int64{entry.ID}, status); err != nil {
		p.logger.Printf("Failed to update entry %d: %v", entry.ID, err)
		stats.Errors++
		return
	}

	p.logger.Printf("Applied action '%s' to entry %d", result.Action, entry.ID)
}

func (p *Processor) replaceEntryText(
	entry *miniflux.Entry,
	feedTitle string,
	rule *Rule,
	stats *ProcessStats,
) {
	entryChanges, changedFields := buildEntryReplacement(entry, rule)
	if len(changedFields) == 0 {
		p.logger.Printf(
			"Rule '%s' matched entry %d but no replaceable text was found in %s",
			rule.Name,
			entry.ID,
			rule.normalizedReplaceField(),
		)
		return
	}

	stats.Replaced++

	if p.dryRun {
		p.logger.Printf(
			"Dry run: would replace substring %q with %q in %s for entry %d [%s] %s",
			rule.ReplaceFrom,
			rule.ReplaceTo,
			strings.Join(changedFields, ", "),
			entry.ID,
			feedTitle,
			entry.Title,
		)
		return
	}

	if _, err := p.client.UpdateEntry(entry.ID, entryChanges); err != nil {
		p.logger.Printf("Failed to replace text for entry %d: %v", entry.ID, err)
		stats.Errors++
		return
	}

	p.logger.Printf(
		"Applied action '%s' to entry %d (%s)",
		actionReplace,
		entry.ID,
		strings.Join(changedFields, ", "),
	)
}

func buildEntryReplacement(
	entry *miniflux.Entry,
	rule *Rule,
) (*miniflux.EntryModificationRequest, []string) {
	entryChanges := &miniflux.EntryModificationRequest{}
	changedFields := make([]string, 0, 2)
	replaceField := rule.normalizedReplaceField()

	if replaceField == replaceFieldTitle || replaceField == replaceFieldBoth {
		updatedTitle := strings.ReplaceAll(entry.Title, rule.ReplaceFrom, rule.ReplaceTo)
		if updatedTitle != entry.Title {
			entryChanges.Title = &updatedTitle
			changedFields = append(changedFields, replaceFieldTitle)
		}
	}

	if replaceField == replaceFieldContent || replaceField == replaceFieldBoth {
		updatedContent := strings.ReplaceAll(entry.Content, rule.ReplaceFrom, rule.ReplaceTo)
		if updatedContent != entry.Content {
			entryChanges.Content = &updatedContent
			changedFields = append(changedFields, replaceFieldContent)
		}
	}

	if len(changedFields) == 0 {
		return nil, nil
	}

	return entryChanges, changedFields
}
