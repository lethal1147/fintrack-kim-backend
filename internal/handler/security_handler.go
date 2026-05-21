package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
	"github.com/joakim/fintrack-api/pkg/response"
)

// SecurityHandler handles /security/* routes.
type SecurityHandler struct {
	svc           domain.SecurityServiceInterface
	refreshSecret string
	cookieSecure  bool
}

func NewSecurityHandler(svc domain.SecurityServiceInterface, refreshSecret string, cookieSecure bool) *SecurityHandler {
	return &SecurityHandler{svc: svc, refreshSecret: refreshSecret, cookieSecure: cookieSecure}
}

func (h *SecurityHandler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// ─── Session types ────────────────────────────────────────────────────────────

type sessionResponse struct {
	ID           string    `json:"id"`
	Device       string    `json:"device"`
	LastActiveAt time.Time `json:"last_active_at"`
	IsCurrent    bool      `json:"is_current"`
}

// ListSessions godoc
// @Summary      List active sessions
// @Description  Returns all active sessions for the authenticated user
// @Tags         security
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/sessions [get]
func (h *SecurityHandler) ListSessions(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)

	currentSessionID := ""
	if cookie, err := c.Cookie(refreshTokenCookie); err == nil {
		if claims, err := jwtutil.ParseRefreshToken(cookie, h.refreshSecret); err == nil {
			currentSessionID = claims.SessionID
		}
	}

	infos, err := h.svc.ListSessions(c.Request.Context(), userID, currentSessionID)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := make([]sessionResponse, len(infos))
	for i, s := range infos {
		resp[i] = sessionResponse{
			ID:           s.ID,
			Device:       s.Device,
			LastActiveAt: s.LastActiveAt,
			IsCurrent:    s.IsCurrent,
		}
	}
	response.Success(c, resp)
}

// RevokeSession godoc
// @Summary      Revoke a session
// @Description  Revokes a single active session by ID
// @Tags         security
// @Param        id  path  string  true  "Session ID"
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/sessions/{id} [delete]
func (h *SecurityHandler) RevokeSession(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	sessionID := c.Param("id")
	if sessionID == "" {
		response.Error(c, apperror.BadRequest("session id required"))
		return
	}
	if err := h.svc.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"revoked": true})
}

// RequestPasswordChange godoc
// @Summary      Request password change OTP
// @Description  Sends a 6-digit OTP to the user's registered email
// @Tags         security
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/password/request [post]
func (h *SecurityHandler) RequestPasswordChange(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	if err := h.svc.RequestPasswordChange(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "code sent"})
}

// ChangePassword godoc
// @Summary      Change password with OTP
// @Description  Verifies OTP and updates the user's password, revoking all sessions
// @Tags         security
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/password/change [post]
func (h *SecurityHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OTP         string `json:"otp"          binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextUserID)
	if err := h.svc.ChangePassword(c.Request.Context(), userID, req.OTP, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}

	h.clearRefreshCookie(c)
	response.Success(c, gin.H{"message": "password changed"})
}

// SetupTOTP godoc
// @Summary      Setup TOTP 2FA
// @Description  Generates a TOTP secret and QR code URI for the authenticated user
// @Tags         security
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/totp/setup [post]
func (h *SecurityHandler) SetupTOTP(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	result, err := h.svc.SetupTOTP(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"secret":       result.Secret,
		"qr_code_uri":  result.QRCodeURI,
		"backup_codes": result.BackupCodes,
	})
}

// ConfirmTOTP godoc
// @Summary      Confirm TOTP setup
// @Description  Validates the TOTP code and enables 2FA, returning 8 backup codes
// @Tags         security
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/totp/confirm [post]
func (h *SecurityHandler) ConfirmTOTP(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextUserID)
	backupCodes, err := h.svc.ConfirmTOTP(c.Request.Context(), userID, req.Code)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"backup_codes": backupCodes})
}

// DisableTOTP godoc
// @Summary      Disable TOTP 2FA
// @Description  Disables 2FA after verifying a TOTP or backup code
// @Tags         security
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /security/totp [delete]
func (h *SecurityHandler) DisableTOTP(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextUserID)
	if err := h.svc.DisableTOTP(c.Request.Context(), userID, req.Code); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"disabled": true})
}
