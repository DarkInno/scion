package auth

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUserToPublic(t *testing.T) {
	now := time.Now()
	u := &User{
		ID:        1,
		Email:     "test@example.com",
		Password:  "secret-hash",
		Name:      "Test",
		CreatedAt: now,
		UpdatedAt: now,
	}
	pub := u.ToPublic()
	if pub.ID != u.ID {
		t.Errorf("ID mismatch: got %d, want %d", pub.ID, u.ID)
	}
	if pub.Email != u.Email {
		t.Errorf("Email mismatch: got %s, want %s", pub.Email, u.Email)
	}
	if pub.Name != u.Name {
		t.Errorf("Name mismatch: got %s, want %s", pub.Name, u.Name)
	}
	if pub.CreatedAt != now.Format(time.RFC3339) {
		t.Errorf("CreatedAt mismatch: got %s, want %s", pub.CreatedAt, now.Format(time.RFC3339))
	}
	// Password must never be exposed
	// We can't test it directly since UserPublic has no Password field,
	// but the struct definition itself enforces this.
}

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: RegisterRequest{
				Email:    "user@example.com",
				Password: "securepassword123",
				Name:     "User",
			},
			wantErr: false,
		},
		{
			name:    "empty email",
			req:     RegisterRequest{Email: "", Password: "password123", Name: "User"},
			wantErr: true,
		},
		{
			name:    "invalid email",
			req:     RegisterRequest{Email: "notanemail", Password: "password123", Name: "User"},
			wantErr: true,
		},
		{
			name:    "short password",
			req:     RegisterRequest{Email: "user@example.com", Password: "short", Name: "User"},
			wantErr: true,
		},
		{
			name:    "long password over 72 bytes",
			req:     RegisterRequest{Email: "user@example.com", Password: strings.Repeat("a", 73), Name: "User"},
			wantErr: true,
		},
		{
			name:    "long email over 254 chars",
			req:     RegisterRequest{Email: strings.Repeat("a", 250) + "@x.com", Password: "password123", Name: "User"},
			wantErr: true,
		},
		{
			name: "password exactly 72 bytes ok",
			req: RegisterRequest{
				Email:    "user@example.com",
				Password: strings.Repeat("a", 72),
				Name:     "User",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLoginRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
	}{
		{
			name:    "valid",
			req:     LoginRequest{Email: "user@example.com", Password: "password123"},
			wantErr: false,
		},
		{
			name:    "empty email",
			req:     LoginRequest{Email: "", Password: "password123"},
			wantErr: true,
		},
		{
			name:    "empty password",
			req:     LoginRequest{Email: "user@example.com", Password: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLoginRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLoginRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User@Example.COM", "user@example.com"},
		{"  user@example.com  ", "user@example.com"},
		{"UPPER@DOMAIN.IO", "upper@domain.io"},
	}

	for _, tt := range tests {
		got := NormalizeEmail(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "X-Real-Ip ignored (security)",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Real-Ip": "10.0.0.1"},
			want:       "192.168.1.1", // X-Real-Ip must NOT be trusted
		},
		{
			name:       "X-Forwarded-For ignored (security)",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.2"},
			want:       "192.168.1.1", // XFF must NOT be trusted
		},
		{
			name:       "X-Forwarded-For multiple ignored (security)",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.3, 10.0.0.4"},
			want:       "192.168.1.1",
		},
		{
			name:       "both headers ignored (security)",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Real-Ip":       "10.0.0.5",
				"X-Forwarded-For": "10.0.0.6",
			},
			want: "192.168.1.1",
		},
		{
			name:       "malformed remote addr fallback",
			remoteAddr: "not-an-addr",
			want:       "not-an-addr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(http.Header),
			}
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			got := ClientIP(req)
			if got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
