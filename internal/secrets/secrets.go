// Package secrets decrypts a local age-encrypted bearer token and gates an
// HTTP handler with it, so the credential this agent needs never has to sit
// in plaintext on disk or in a cloud secrets manager.
package secrets

import (
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"filippo.io/age"
)

// LoadIdentity reads an age identity (private key) from a key file in the
// standard "AGE-SECRET-KEY-1..." text format produced by `age-keygen`.
func LoadIdentity(path string) (age.Identity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ids, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("secrets: parsing identity file %s: %w", path, err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("secrets: no identities found in %s", path)
	}
	return ids[0], nil
}

// DecryptBearerToken decrypts the age-encrypted file at path with identity
// and returns its contents as a bearer token, with surrounding whitespace
// trimmed.
func DecryptBearerToken(path string, identity age.Identity) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	r, err := age.Decrypt(f, identity)
	if err != nil {
		return "", fmt.Errorf("secrets: decrypting %s: %w", path, err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// RequireBearerToken wraps next so a request only reaches it when its
// Authorization header is exactly "Bearer <token>". The comparison runs in
// constant time to avoid leaking the token through response-time timing.
func RequireBearerToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
