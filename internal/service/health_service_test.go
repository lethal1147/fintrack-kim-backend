package service

import (
	"errors"
	"testing"
)

type mockPinger struct{ err error }

func (m *mockPinger) Ping() error { return m.err }

func TestHealthService_DBUp(t *testing.T) {
	svc := NewHealthService("0.1.0", &mockPinger{})
	result := svc.Check()
	if result.Status != "ok" {
		t.Errorf("want status=ok, got %s", result.Status)
	}
	if result.DB != "ok" {
		t.Errorf("want db=ok, got %s", result.DB)
	}
	if result.Version != "0.1.0" {
		t.Errorf("want version=0.1.0, got %s", result.Version)
	}
}

func TestHealthService_DBDown(t *testing.T) {
	svc := NewHealthService("0.1.0", &mockPinger{err: errors.New("conn refused")})
	result := svc.Check()
	if result.Status != "ok" {
		t.Errorf("want status=ok, got %s", result.Status)
	}
	if result.DB != "error" {
		t.Errorf("want db=error, got %s", result.DB)
	}
}
