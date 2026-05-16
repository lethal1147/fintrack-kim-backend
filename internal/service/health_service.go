package service

import "github.com/joakim/fintrack-api/pkg/response"

type DBPinger interface {
	Ping() error
}

type HealthService struct {
	version string
	pinger  DBPinger
}

func NewHealthService(version string, pinger DBPinger) *HealthService {
	return &HealthService{version: version, pinger: pinger}
}

func (s *HealthService) Check() response.HealthResponse {
	dbStatus := "ok"
	if err := s.pinger.Ping(); err != nil {
		dbStatus = "error"
	}
	return response.HealthResponse{
		Status:  "ok",
		Version: s.version,
		DB:      dbStatus,
	}
}
