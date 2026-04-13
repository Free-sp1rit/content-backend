package service

import (
	"errors"
	"testing"
)

func assertErrIs(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got err %v, want %v", got, want)
	}
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hash password for test setup: %v", err)
	}

	return hash
}

func assertPasswordMatches(t *testing.T, hash, password string) {
	t.Helper()

	if err := comparePassword(hash, password); err != nil {
		t.Fatalf("compare password hash: %v", err)
	}
}
