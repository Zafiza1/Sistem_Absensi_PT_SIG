package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

func TestGenerateAndParseAccessToken_RoundTrip(t *testing.T) {
	manager := NewManager("test-secret")
	userID := uuid.New()

	token, err := manager.GenerateAccessToken(userID, rbac.Admin, "Test Admin", time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("claims.Subject = %q, want %q", claims.Subject, userID.String())
	}
	if claims.Role != rbac.Admin {
		t.Errorf("claims.Role = %q, want %q", claims.Role, rbac.Admin)
	}
	if claims.Name != "Test Admin" {
		t.Errorf("claims.Name = %q, want %q", claims.Name, "Test Admin")
	}
}

func TestParseAccessToken_Expired(t *testing.T) {
	manager := NewManager("test-secret")
	token, err := manager.GenerateAccessToken(uuid.New(), rbac.HR, "Test HR", -time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if _, err := manager.ParseAccessToken(token); err == nil {
		t.Fatal("ParseAccessToken() error = nil, want an expiry error")
	}
}

func TestParseAccessToken_WrongSecret(t *testing.T) {
	issuer := NewManager("secret-a")
	verifier := NewManager("secret-b")

	token, err := issuer.GenerateAccessToken(uuid.New(), rbac.Management, "Test Manager", time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if _, err := verifier.ParseAccessToken(token); err == nil {
		t.Fatal("ParseAccessToken() error = nil, want a signature error")
	}
}

func TestParseAccessToken_Garbage(t *testing.T) {
	manager := NewManager("test-secret")
	if _, err := manager.ParseAccessToken("not-a-valid-token"); err == nil {
		t.Fatal("ParseAccessToken() error = nil, want an error for garbage input")
	}
}
