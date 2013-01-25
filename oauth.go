// Copyright 2013 Canonical Ltd.  This software is licensed under the
// GNU Lesser General Public License version 3 (see the file COPYING).

package gomaasapi

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func init() {
	// Initialize the random generator.
	rand.Seed(time.Now().UTC().UnixNano())
}

func generateNonce() string {
	return strconv.Itoa(rand.Intn(100000000))
}

func generateTimestamp() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

type OAuthSigner interface {
	OAuthSign(request *http.Request) error
}

type OAuthToken struct {
	ConsumerKey    string
	ConsumerSecret string
	TokenKey       string
	TokenSecret    string
}

// Trick to ensure *_PLAINTEXTOAuthSigner implements the OAuthSigner interface.
var _ OAuthSigner = (*_PLAINTEXTOAuthSigner)(nil)

type _PLAINTEXTOAuthSigner struct {
	token *OAuthToken
	realm string
}

func NewPLAINTEXTOAuthSigner(token *OAuthToken, realm string) (OAuthSigner, error) {
	return _PLAINTEXTOAuthSigner{token, realm}, nil
}

// OAuthSignPLAINTEXT signs the provided request using the OAuth PLAINTEXT
// method: http://oauth.net/core/1.0/#anchor22.
func (signer _PLAINTEXTOAuthSigner) OAuthSign(request *http.Request) error {

	signature := signer.token.ConsumerSecret + `&` + signer.token.TokenSecret
	authData := map[string]string{
		"realm":                  signer.realm,
		"oauth_consumer_key":     signer.token.ConsumerKey,
		"oauth_token":            signer.token.TokenKey,
		"oauth_signature_method": "PLAINTEXT",
		"oauth_signature":        signature,
		"oauth_timestamp":        generateTimestamp(),
		"oauth_nonce":            generateNonce(),
		"oauth_version":          "1.0",
	}
	// Build OAuth header.
	authHeader := []string{}
	for key, value := range authData {
		authHeader = append(authHeader, fmt.Sprintf(`%s="%s"`, key, url.QueryEscape(value)))
	}
	strHeader := "OAuth " + strings.Join(authHeader, ", ")
	request.Header.Add("Authorization", strHeader)
	return nil
}
