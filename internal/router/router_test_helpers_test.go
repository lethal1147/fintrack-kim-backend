package router

import (
	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/handler"
	"github.com/joakim/fintrack-api/internal/service"
)

type stubAuthSvc struct{}

func (s *stubAuthSvc) Register(_ service.AuthInput) (*service.AuthResponse, error)    { return nil, nil }
func (s *stubAuthSvc) Login(_ service.LoginInput) (*service.AuthResponse, error)      { return nil, nil }
func (s *stubAuthSvc) Refresh(_ string) (*service.RefreshResponse, error)             { return nil, nil }
func (s *stubAuthSvc) Logout(_ string) error                                          { return nil }
func (s *stubAuthSvc) LogoutAll(_ string) error                                       { return nil }
func (s *stubAuthSvc) GetProfile(_ string) (*service.UserInfo, error)                 { return nil, nil }

// newTestRouter wires stub handlers into the real router.New for integration tests.
func newTestRouter(cfg RouterConfig) *gin.Engine {
	healthHandler := handler.NewHealthHandler(&stubHealthSvc{})
	authHandler := handler.NewAuthHandler(&stubAuthSvc{})
	return New(cfg, Handlers{Health: healthHandler, Auth: authHandler})
}
