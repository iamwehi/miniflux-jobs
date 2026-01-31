package main

import (
	"regexp"
	"strings"

	miniflux "miniflux.app/v2/client"
)

// Matcher handles rule matching against entries
type Matcher struct {
	compiledRules []compiledRule
}

// compiledRule holds pre-compiled regex patterns for a rule
type compiledRule struct {
	rule    Rule
	feed    *regexp.Regexp
	author  *regexp.Regexp
	title   *regexp.Regexp
	content *regexp.Regexp
}

// NewMatcher creates a new Matcher with pre-compiled regex patterns
func NewMatcher(rules []Rule) (*Matcher, error) {
	compiled := make([]compiledRule, 0, len(rules))

	for _, rule := range rules {
		cr := compiledRule{rule: rule}
		var err error

		if rule.Feed != "" {
			cr.feed, err = regexp.Compile(rule.Feed)
			if err != nil {
				return nil, &RegexError{Field: "feed", Rule: rule.Name, Err: err}
			}
		}

		if rule.Author != "" {
			cr.author, err = regexp.Compile(rule.Author)
			if err != nil {
				return nil, &RegexError{Field: "author", Rule: rule.Name, Err: err}
			}
		}

		if rule.Title != "" {
			cr.title, err = regexp.Compile(rule.Title)
			if err != nil {
				return nil, &RegexError{Field: "title", Rule: rule.Name, Err: err}
			}
		}

		if rule.Content != "" {
			cr.content, err = regexp.Compile(rule.Content)
			if err != nil {
				return nil, &RegexError{Field: "content", Rule: rule.Name, Err: err}
			}
		}

		compiled = append(compiled, cr)
	}

	return &Matcher{compiledRules: compiled}, nil
}

// RegexError represents an error in compiling a regex pattern
type RegexError struct {
	Field string
	Rule  string
	Err   error
}

func (e *RegexError) Error() string {
	return "invalid regex in rule '" + e.Rule + "' field '" + e.Field + "': " + e.Err.Error()
}

// MatchResult contains the result of matching an entry against rules
type MatchResult struct {
	Matched bool
	Rule    *Rule
	Action  string // normalized action: "read" or "remove"
}

// Match checks if an entry matches any rule and returns the first matching rule
func (m *Matcher) Match(entry *miniflux.Entry) MatchResult {
	for _, cr := range m.compiledRules {
		if m.matchRule(entry, &cr) {
			return MatchResult{
				Matched: true,
				Rule:    &cr.rule,
				Action:  strings.ToLower(cr.rule.Action),
			}
		}
	}
	return MatchResult{Matched: false}
}

// matchRule checks if an entry matches a single compiled rule
// All non-empty patterns must match (AND logic)
func (m *Matcher) matchRule(entry *miniflux.Entry, cr *compiledRule) bool {
	// Check feed title
	if cr.feed != nil {
		feedTitle := ""
		if entry.Feed != nil {
			feedTitle = entry.Feed.Title
		}
		if !cr.feed.MatchString(feedTitle) {
			return false
		}
	}

	// Check author
	if cr.author != nil {
		if !cr.author.MatchString(entry.Author) {
			return false
		}
	}

	// Check entry title
	if cr.title != nil {
		if !cr.title.MatchString(entry.Title) {
			return false
		}
	}

	// Check content
	if cr.content != nil {
		if !cr.content.MatchString(entry.Content) {
			return false
		}
	}

	return true
}
