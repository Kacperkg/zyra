package config

import "testing"

func TestDevelopmentAdminRequiresExplicitLocalMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "unused")
	t.Setenv("JWT_SECRET", "test-secret-of-at-least-32-characters")
	t.Setenv("BOOTSTRAP_ADMIN_EMAIL", "")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")
	for _, tc := range []struct {
		name, enabled, env, address, container string
		valid                                  bool
	}{
		{"off", "", "", "0.0.0.0:8080", "", true},
		{"local", "true", "development", "127.0.0.1:8080", "", true},
		{"ipv6", "true", "development", "[::1]:8080", "", true},
		{"public", "true", "development", "0.0.0.0:8080", "", false},
		{"container", "true", "development", "0.0.0.0:8080", "true", true},
		{"container-specific-address", "true", "development", "192.0.2.1:8080", "true", false},
		{"production-container", "true", "production", "0.0.0.0:8080", "true", false},
		{"production", "true", "production", "127.0.0.1:8080", "", false},
		{"unspecified", "true", "", "127.0.0.1:8080", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DEV_ADMIN_LOGIN", tc.enabled)
			t.Setenv("DEV_ADMIN_CONTAINER", tc.container)
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("ADDRESS", tc.address)
			_, err := Load()
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}
