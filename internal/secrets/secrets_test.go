package secrets

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
)

// writeEncryptedFixture generates a fresh age identity, encrypts token to
// its recipient, and writes both the identity key file and the encrypted
// token file into dir. It returns their paths.
func writeEncryptedFixture(t *testing.T, dir, token string) (identityPath, secretsPath string) {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}

	identityPath = filepath.Join(dir, "key.txt")
	if err := os.WriteFile(identityPath, []byte(identity.String()+"\n"), 0o600); err != nil {
		t.Fatalf("writing identity file: %v", err)
	}

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, identity.Recipient())
	if err != nil {
		t.Fatalf("age.Encrypt: %v", err)
	}
	if _, err := w.Write([]byte(token)); err != nil {
		t.Fatalf("writing plaintext: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing age writer: %v", err)
	}

	secretsPath = filepath.Join(dir, "secrets.txt.age")
	if err := os.WriteFile(secretsPath, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("writing secrets file: %v", err)
	}

	return identityPath, secretsPath
}

func TestLoadIdentityAndDecryptBearerToken(t *testing.T) {
	dir := t.TempDir()
	identityPath, secretsPath := writeEncryptedFixture(t, dir, "s3cr3t-token\n")

	identity, err := LoadIdentity(identityPath)
	if err != nil {
		t.Fatalf("LoadIdentity() error = %v", err)
	}

	token, err := DecryptBearerToken(secretsPath, identity)
	if err != nil {
		t.Fatalf("DecryptBearerToken() error = %v", err)
	}
	if token != "s3cr3t-token" {
		t.Errorf("token = %q, want %q (trimmed)", token, "s3cr3t-token")
	}
}

func TestDecryptBearerTokenWrongIdentity(t *testing.T) {
	dir := t.TempDir()
	_, secretsPath := writeEncryptedFixture(t, dir, "s3cr3t-token")

	otherIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}

	if _, err := DecryptBearerToken(secretsPath, otherIdentity); err == nil {
		t.Error("DecryptBearerToken() with the wrong identity: want error, got nil")
	}
}

func TestRequireBearerToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireBearerToken("correct-token", inner)

	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"correct token", "Bearer correct-token", http.StatusOK},
		{"missing header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer wrong-token", http.StatusUnauthorized},
		{"missing Bearer prefix", "correct-token", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}
