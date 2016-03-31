// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package gomaasapi

import "net/url"

// URLParams wraps url.Values to easily add values, but skipping empty ones.
type URLParams struct {
	Values url.Values
}

// NewURLParams allocates a new URLParams type.
func NewURLParams() *URLParams {
	return &URLParams{Values: make(url.Values)}
}

// MaybeAdd adds the (name, value) pair iff value is not empty.
func (p *URLParams) MaybeAdd(name, value string) {
	if value != "" {
		p.Values.Add(name, value)
	}
}

// MaybeAddMany adds the (name, value) for each value in values iff
// value is not empty.
func (p *URLParams) MaybeAddMany(name string, values []string) {
	for _, value := range values {
		p.MaybeAdd(name, value)
	}
}
