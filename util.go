// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"strings"
)

// JoinURLs joins a base URL and a subpath together.
// Regardless of whether baseURL ends in a trailing slash (or even multiple
// trailing slashes), or whether there are any leading slashes at the begining
// of path, the two will always be joined together by a single slash.
func JoinURLs(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}
