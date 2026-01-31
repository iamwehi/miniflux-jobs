package main

import (
	miniflux "miniflux.app/v2/client"
)

// MinifluxClient defines the interface for interacting with Miniflux API
// This interface allows for easy mocking in tests
type MinifluxClient interface {
	Entries(filter *miniflux.Filter) (*miniflux.EntryResultSet, error)
	UpdateEntries(entryIDs []int64, status string) error
	Feeds() (miniflux.Feeds, error)
}

// ClientWrapper wraps the actual Miniflux client to implement MinifluxClient interface
type ClientWrapper struct {
	client *miniflux.Client
}

// NewClientWrapper creates a new ClientWrapper with the given Miniflux client
func NewClientWrapper(endpoint, apiKey string) *ClientWrapper {
	client := miniflux.NewClient(endpoint, apiKey)
	return &ClientWrapper{client: client}
}

// Entries fetches entries from Miniflux with the given filter
func (c *ClientWrapper) Entries(filter *miniflux.Filter) (*miniflux.EntryResultSet, error) {
	return c.client.Entries(filter)
}

// UpdateEntries updates the status of the given entries
func (c *ClientWrapper) UpdateEntries(entryIDs []int64, status string) error {
	return c.client.UpdateEntries(entryIDs, status)
}

// Feeds fetches all feeds from Miniflux
func (c *ClientWrapper) Feeds() (miniflux.Feeds, error) {
	return c.client.Feeds()
}
