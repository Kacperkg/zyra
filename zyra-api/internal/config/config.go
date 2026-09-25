package config

import (
	"errors"
	"net"
	"os"
	"strings"
)

type Config struct {
	Address, DatabaseURL, JWTSecret, BootstrapEmail, BootstrapPassword string
	Migrate                                                            bool
	DevAdminLogin                                                      bool
}

func Load() (Config, error) {
	c := Config{Address: os.Getenv("ADDRESS"), DatabaseURL: os.Getenv("DATABASE_URL"), JWTSecret: os.Getenv("JWT_SECRET"), BootstrapEmail: os.Getenv("BOOTSTRAP_ADMIN_EMAIL"), BootstrapPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"), Migrate: os.Getenv("AUTO_MIGRATE") == "true"}
	if c.Address == "" {
		c.Address = "127.0.0.1:8080"
	}
	c.DevAdminLogin = os.Getenv("DEV_ADMIN_LOGIN") == "true"
	if c.DevAdminLogin {
		host, _, err := net.SplitHostPort(c.Address)
		ip := net.ParseIP(host)
		if os.Getenv("APP_ENV") != "development" || err != nil || ip == nil || !ip.IsLoopback() {
			return c, errors.New("DEV_ADMIN_LOGIN requires APP_ENV=development and a loopback IP listener")
		}
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	if len(strings.TrimSpace(c.JWTSecret)) < 32 {
		return c, errors.New("JWT_SECRET must contain at least 32 characters")
	}
	if (c.BootstrapEmail == "") != (c.BootstrapPassword == "") {
		return c, errors.New("both bootstrap admin variables are required together")
	}
	return c, nil
}
