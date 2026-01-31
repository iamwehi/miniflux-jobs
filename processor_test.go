package main

import (
	"log"
	"os"
	"testing"

	miniflux "miniflux.app/v2/client"
)

// MockClient implements MinifluxClient for testing
type MockClient struct {
	entries       []*miniflux.Entry
	updatedIDs    []int64
	updatedStatus string
	feeds         miniflux.Feeds
	entriesErr    error
	updateErr     error
	feedsErr      error
}

func (m *MockClient) Entries(filter *miniflux.Filter) (*miniflux.EntryResultSet, error) {
	if m.entriesErr != nil {
		return nil, m.entriesErr
	}

	// Apply offset and limit
	start := filter.Offset
	if start >= len(m.entries) {
		return &miniflux.EntryResultSet{
			Total:   len(m.entries),
			Entries: []*miniflux.Entry{},
		}, nil
	}

	end := start + filter.Limit
	if end > len(m.entries) || filter.Limit == 0 {
		end = len(m.entries)
	}

	return &miniflux.EntryResultSet{
		Total:   len(m.entries),
		Entries: m.entries[start:end],
	}, nil
}

func (m *MockClient) UpdateEntries(entryIDs []int64, status string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updatedIDs = append(m.updatedIDs, entryIDs...)
	m.updatedStatus = status
	return nil
}

func (m *MockClient) Feeds() (miniflux.Feeds, error) {
	if m.feedsErr != nil {
		return nil, m.feedsErr
	}
	return m.feeds, nil
}

func TestProcessorMarkRead(t *testing.T) {
	mockClient := &MockClient{
		entries: []*miniflux.Entry{
			{
				ID:      1,
				Title:   "Sponsored Post",
				Author:  "Bob",
				Content: "Buy now!",
				Feed:    &miniflux.Feed{Title: "Tech News"},
			},
			{
				ID:      2,
				Title:   "Regular Post",
				Author:  "Alice",
				Content: "Normal content",
				Feed:    &miniflux.Feed{Title: "Tech News"},
			},
		},
	}

	rules := []Rule{
		{
			Name:   "Mark sponsored as read",
			Title:  "(?i)sponsored",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	logger := log.New(os.Stdout, "[test] ", 0)
	processor := NewProcessor(mockClient, matcher, logger)

	stats, err := processor.Process()
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", stats.TotalEntries)
	}
	if stats.MatchedEntries != 1 {
		t.Errorf("Expected 1 matched entry, got %d", stats.MatchedEntries)
	}
	if stats.MarkedRead != 1 {
		t.Errorf("Expected 1 marked read, got %d", stats.MarkedRead)
	}
	if stats.Removed != 0 {
		t.Errorf("Expected 0 removed, got %d", stats.Removed)
	}

	if len(mockClient.updatedIDs) != 1 || mockClient.updatedIDs[0] != 1 {
		t.Errorf("Expected entry 1 to be updated, got %v", mockClient.updatedIDs)
	}
	if mockClient.updatedStatus != miniflux.EntryStatusRead {
		t.Errorf("Expected status 'read', got '%s'", mockClient.updatedStatus)
	}
}

func TestProcessorRemove(t *testing.T) {
	mockClient := &MockClient{
		entries: []*miniflux.Entry{
			{
				ID:      1,
				Title:   "Promo Post",
				Author:  "Bob",
				Content: "#promo content",
				Feed:    &miniflux.Feed{Title: "Tech News"},
			},
		},
	}

	rules := []Rule{
		{
			Name:    "Remove Bob's promos",
			Author:  "Bob",
			Content: "#promo",
			Action:  "remove",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	logger := log.New(os.Stdout, "[test] ", 0)
	processor := NewProcessor(mockClient, matcher, logger)

	stats, err := processor.Process()
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if stats.Removed != 1 {
		t.Errorf("Expected 1 removed, got %d", stats.Removed)
	}

	if mockClient.updatedStatus != miniflux.EntryStatusRemoved {
		t.Errorf("Expected status 'removed', got '%s'", mockClient.updatedStatus)
	}
}

func TestProcessorNoMatches(t *testing.T) {
	mockClient := &MockClient{
		entries: []*miniflux.Entry{
			{
				ID:      1,
				Title:   "Regular Post",
				Author:  "Alice",
				Content: "Normal content",
				Feed:    &miniflux.Feed{Title: "Tech News"},
			},
		},
	}

	rules := []Rule{
		{
			Name:   "Match Bob only",
			Author: "Bob",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	logger := log.New(os.Stdout, "[test] ", 0)
	processor := NewProcessor(mockClient, matcher, logger)

	stats, err := processor.Process()
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 total entry, got %d", stats.TotalEntries)
	}
	if stats.MatchedEntries != 0 {
		t.Errorf("Expected 0 matched entries, got %d", stats.MatchedEntries)
	}
	if len(mockClient.updatedIDs) != 0 {
		t.Errorf("Expected no updates, got %v", mockClient.updatedIDs)
	}
}

func TestProcessorEmptyEntries(t *testing.T) {
	mockClient := &MockClient{
		entries: []*miniflux.Entry{},
	}

	rules := []Rule{
		{
			Name:   "Match anything",
			Author: ".*",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	logger := log.New(os.Stdout, "[test] ", 0)
	processor := NewProcessor(mockClient, matcher, logger)

	stats, err := processor.Process()
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if stats.TotalEntries != 0 {
		t.Errorf("Expected 0 total entries, got %d", stats.TotalEntries)
	}
}

func TestProcessorMultipleRules(t *testing.T) {
	mockClient := &MockClient{
		entries: []*miniflux.Entry{
			{
				ID:      1,
				Title:   "Sponsored Post",
				Author:  "Bob",
				Content: "Content",
				Feed:    &miniflux.Feed{Title: "Tech News"},
			},
			{
				ID:      2,
				Title:   "Promo Post",
				Author:  "Alice",
				Content: "#promo",
				Feed:    &miniflux.Feed{Title: "Sports"},
			},
			{
				ID:      3,
				Title:   "Regular Post",
				Author:  "Charlie",
				Content: "Normal",
				Feed:    &miniflux.Feed{Title: "News"},
			},
		},
	}

	rules := []Rule{
		{
			Name:   "Mark sponsored as read",
			Title:  "(?i)sponsored",
			Action: "read",
		},
		{
			Name:    "Remove promos",
			Content: "#promo",
			Action:  "remove",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	logger := log.New(os.Stdout, "[test] ", 0)
	processor := NewProcessor(mockClient, matcher, logger)

	stats, err := processor.Process()
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats.TotalEntries)
	}
	if stats.MatchedEntries != 2 {
		t.Errorf("Expected 2 matched entries, got %d", stats.MatchedEntries)
	}
	if stats.MarkedRead != 1 {
		t.Errorf("Expected 1 marked read, got %d", stats.MarkedRead)
	}
	if stats.Removed != 1 {
		t.Errorf("Expected 1 removed, got %d", stats.Removed)
	}
}

func TestProcessorPagination(t *testing.T) {
	// Create 150 entries to test pagination (batch size is 100)
	entries := make([]*miniflux.Entry, 150)
	for i := 0; i < 150; i++ {
		entries[i] = &miniflux.Entry{
			ID:      int64(i + 1),
			Title:   "Test Post",
			Author:  "Bob",
			Content: "Content",
		}
	}

	mockClient := &MockClient{
		entries: entries,
	}

	rules := []Rule{
		{
			Name:   "Match all Bob",
			Author: "Bob",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	logger := log.New(os.Stdout, "[test] ", 0)
	processor := NewProcessor(mockClient, matcher, logger)

	stats, err := processor.Process()
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if stats.TotalEntries != 150 {
		t.Errorf("Expected 150 total entries, got %d", stats.TotalEntries)
	}
	if stats.MatchedEntries != 150 {
		t.Errorf("Expected 150 matched entries, got %d", stats.MatchedEntries)
	}
}
