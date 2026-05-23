package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/handler"
	"github.com/joakim/fintrack-api/internal/service"
)

type stubAuthSvc struct{}

// stubSessionRepo satisfies domain.SessionRepository for router tests.
type stubSessionRepo struct{}

func (s *stubSessionRepo) Create(_ *domain.Session) error                              { return nil }
func (s *stubSessionRepo) FindByID(_ string) (*domain.Session, error)                  { return nil, nil }
func (s *stubSessionRepo) FindByRefreshToken(_ string) (*domain.Session, error)        { return nil, nil }
func (s *stubSessionRepo) DeleteByID(_ string) error                                   { return nil }
func (s *stubSessionRepo) DeleteAllByUserID(_ string) error                            { return nil }
func (s *stubSessionRepo) ListByUserID(_ string) ([]*domain.Session, error)            { return nil, nil }
func (s *stubSessionRepo) UpdateLastActive(_ string, _ time.Time) error                { return nil }

func (s *stubAuthSvc) Register(_ service.AuthInput) (*service.AuthResponse, error)         { return nil, nil }
func (s *stubAuthSvc) Login(_ service.LoginInput) (*service.LoginResult, error)            { return nil, nil }
func (s *stubAuthSvc) Refresh(_ string) (*service.RefreshResponse, error)                  { return nil, nil }
func (s *stubAuthSvc) Logout(_ string) error                                               { return nil }
func (s *stubAuthSvc) LogoutAll(_ string) error                                            { return nil }
func (s *stubAuthSvc) GetProfile(_ string) (*service.UserInfo, error)                      { return nil, nil }
func (s *stubAuthSvc) VerifyTOTP(_, _, _, _ string) (*service.AuthResponse, error)         { return nil, nil }

type stubTxSvc struct{}

func (s *stubTxSvc) Create(_ string, _ domain.CreateTransactionRequest) (*domain.Transaction, error) {
	return nil, nil
}
func (s *stubTxSvc) Get(_, _ string) (*domain.Transaction, error)  { return nil, nil }
func (s *stubTxSvc) Update(_, _ string, _ domain.UpdateTransactionRequest) (*domain.Transaction, error) {
	return nil, nil
}
func (s *stubTxSvc) Delete(_, _ string) error { return nil }
func (s *stubTxSvc) List(_ string, _ domain.TransactionFilter) (*domain.TransactionListResult, error) {
	return &domain.TransactionListResult{}, nil
}
func (s *stubTxSvc) Summary(_ string, _, _ time.Time) (*domain.TransactionSummaryResult, error) {
	return &domain.TransactionSummaryResult{}, nil
}

// newTestRouter wires stub handlers into the real router.New for integration tests.
func newTestRouter(cfg RouterConfig) *gin.Engine {
	if cfg.SessionRepo == nil {
		cfg.SessionRepo = &stubSessionRepo{}
	}
	healthHandler := handler.NewHealthHandler(&stubHealthSvc{})
	authHandler := handler.NewAuthHandler(&stubAuthSvc{}, false)
	txHandler := handler.NewTransactionHandler(&stubTxSvc{})
	return New(cfg, Handlers{Health: healthHandler, Auth: authHandler, Transaction: txHandler})
}
