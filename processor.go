package main

import (
	"fmt"
	"log"

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
	case "read":
		status = miniflux.EntryStatusRead
		stats.MarkedRead++
	case "remove":
		status = miniflux.EntryStatusRemoved
		stats.Removed++
	default:
		p.logger.Printf("Unknown action '%s' for rule '%s'", result.Action, result.Rule.Name)
		stats.Errors++
		return
	}

	if p.dryRun {
		actionVerb := result.Action
		if result.Action == "read" {
			actionVerb = "mark read"
		} else if result.Action == "remove" {
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
