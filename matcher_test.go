package main

import (
	"testing"

	miniflux "miniflux.app/v2/client"
)

func TestMatcherSimpleMatch(t *testing.T) {
	rules := []Rule{
		{
			Name:    "Match Bob's promos",
			Author:  "Bob",
			Content: "#promo",
			Action:  "remove",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	entry := &miniflux.Entry{
		ID:      1,
		Title:   "Test Post",
		Author:  "Bob",
		Content: "This is a #promo post",
		Feed:    &miniflux.Feed{Title: "Tech News"},
	}

	result := matcher.Match(entry)
	if !result.Matched {
		t.Error("Expected entry to match")
	}
	if result.Action != "remove" {
		t.Errorf("Expected action 'remove', got '%s'", result.Action)
	}
	if result.Rule.Name != "Match Bob's promos" {
		t.Errorf("Expected rule name 'Match Bob's promos', got '%s'", result.Rule.Name)
	}
}

func TestMatcherNoMatch(t *testing.T) {
	rules := []Rule{
		{
			Name:    "Match Bob's promos",
			Author:  "Bob",
			Content: "#promo",
			Action:  "remove",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	entry := &miniflux.Entry{
		ID:      1,
		Title:   "Test Post",
		Author:  "Alice",
		Content: "This is a regular post",
		Feed:    &miniflux.Feed{Title: "Tech News"},
	}

	result := matcher.Match(entry)
	if result.Matched {
		t.Error("Expected entry not to match")
	}
}

func TestMatcherRegexMatch(t *testing.T) {
	rules := []Rule{
		{
			Name:   "Match sponsored titles",
			Title:  "(?i)sponsored|advertisement",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	testCases := []struct {
		title    string
		expected bool
	}{
		{"SPONSORED: Great Product", true},
		{"sponsored content", true},
		{"This is an Advertisement", true},
		{"Regular post title", false},
	}

	for _, tc := range testCases {
		entry := &miniflux.Entry{
			ID:    1,
			Title: tc.title,
		}
		result := matcher.Match(entry)
		if result.Matched != tc.expected {
			t.Errorf("Title '%s': expected matched=%v, got matched=%v", tc.title, tc.expected, result.Matched)
		}
	}
}

func TestMatcherFeedMatch(t *testing.T) {
	rules := []Rule{
		{
			Name:   "Match Tech feeds",
			Feed:   "Tech.*",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	testCases := []struct {
		feedTitle string
		expected  bool
	}{
		{"Tech News", true},
		{"TechCrunch", true},
		{"Sports News", false},
	}

	for _, tc := range testCases {
		entry := &miniflux.Entry{
			ID:    1,
			Title: "Test",
			Feed:  &miniflux.Feed{Title: tc.feedTitle},
		}
		result := matcher.Match(entry)
		if result.Matched != tc.expected {
			t.Errorf("Feed '%s': expected matched=%v, got matched=%v", tc.feedTitle, tc.expected, result.Matched)
		}
	}
}

func TestMatcherNilFeed(t *testing.T) {
	rules := []Rule{
		{
			Name:   "Match Tech feeds",
			Feed:   "Tech.*",
			Action: "read",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	entry := &miniflux.Entry{
		ID:    1,
		Title: "Test",
		Feed:  nil, // No feed info
	}

	result := matcher.Match(entry)
	if result.Matched {
		t.Error("Expected entry with nil feed not to match feed pattern")
	}
}

func TestMatcherANDLogic(t *testing.T) {
	rules := []Rule{
		{
			Name:    "Match Bob's promos in Tech News",
			Feed:    "Tech News",
			Author:  "Bob",
			Content: "#promo",
			Action:  "remove",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	// All conditions match
	entry1 := &miniflux.Entry{
		ID:      1,
		Author:  "Bob",
		Content: "This is a #promo post",
		Feed:    &miniflux.Feed{Title: "Tech News"},
	}
	if !matcher.Match(entry1).Matched {
		t.Error("Expected entry1 to match (all conditions met)")
	}

	// Author doesn't match
	entry2 := &miniflux.Entry{
		ID:      2,
		Author:  "Alice",
		Content: "This is a #promo post",
		Feed:    &miniflux.Feed{Title: "Tech News"},
	}
	if matcher.Match(entry2).Matched {
		t.Error("Expected entry2 not to match (wrong author)")
	}

	// Content doesn't match
	entry3 := &miniflux.Entry{
		ID:      3,
		Author:  "Bob",
		Content: "Regular post",
		Feed:    &miniflux.Feed{Title: "Tech News"},
	}
	if matcher.Match(entry3).Matched {
		t.Error("Expected entry3 not to match (no #promo)")
	}

	// Feed doesn't match
	entry4 := &miniflux.Entry{
		ID:      4,
		Author:  "Bob",
		Content: "This is a #promo post",
		Feed:    &miniflux.Feed{Title: "Sports News"},
	}
	if matcher.Match(entry4).Matched {
		t.Error("Expected entry4 not to match (wrong feed)")
	}
}

func TestMatcherInvalidRegex(t *testing.T) {
	rules := []Rule{
		{
			Name:   "Invalid regex",
			Feed:   "[invalid",
			Action: "read",
		},
	}

	_, err := NewMatcher(rules)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}

	regexErr, ok := err.(*RegexError)
	if !ok {
		t.Errorf("Expected RegexError, got %T", err)
	}
	if regexErr.Field != "feed" {
		t.Errorf("Expected field 'feed', got '%s'", regexErr.Field)
	}
}

func TestMatcherFirstRuleWins(t *testing.T) {
	rules := []Rule{
		{
			Name:   "First rule",
			Author: "Bob",
			Action: "read",
		},
		{
			Name:   "Second rule",
			Author: "Bob",
			Action: "remove",
		},
	}

	matcher, err := NewMatcher(rules)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	entry := &miniflux.Entry{
		ID:     1,
		Author: "Bob",
	}

	result := matcher.Match(entry)
	if !result.Matched {
		t.Error("Expected entry to match")
	}
	if result.Rule.Name != "First rule" {
		t.Errorf("Expected first rule to match, got '%s'", result.Rule.Name)
	}
	if result.Action != "read" {
		t.Errorf("Expected action 'read', got '%s'", result.Action)
	}
}

func TestMatcherEmptyRules(t *testing.T) {
	matcher, err := NewMatcher([]Rule{})
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	entry := &miniflux.Entry{
		ID:    1,
		Title: "Test",
	}

	result := matcher.Match(entry)
	if result.Matched {
		t.Error("Expected no match with empty rules")
	}
}
