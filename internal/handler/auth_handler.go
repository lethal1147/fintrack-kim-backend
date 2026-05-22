package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/internal/service"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/response"
)

const refreshTokenCookie = "refresh_token"

type AuthHandler struct {
	svc          service.AuthServiceInterface
	cookieSecure bool
}

// NewAuthHandler creates an AuthHandler. cookieSecure should be true in production (HTTPS).
func NewAuthHandler(svc service.AuthServiceInterface, cookieSecure bool) *AuthHandler {
	return &AuthHandler{svc: svc, cookieSecure: cookieSecure}
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"     binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60,
	})
}

func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
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

// Register godoc
// @Summary      Register a new user
// @Description  Creates a local user account. Sets refresh_token as an httpOnly cookie and returns access_token + user in the body.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      registerRequest  true  "Registration payload"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      409   {object}  map[string]interface{}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.Register(service.AuthInput{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	h.setRefreshCookie(c, resp.RefreshToken)
	response.Created(c, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
	})
}

// Login godoc
// @Summary      Login
// @Description  Verifies credentials. Sets refresh_token as an httpOnly cookie and returns access_token + user in the body.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      loginRequest  true  "Login payload"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	result, err := h.svc.Login(service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	if result.Challenge != nil {
		response.Success(c, gin.H{
			"totp_required":   result.Challenge.TOTPRequired,
			"challenge_token": result.Challenge.ChallengeToken,
		})
		return
	}

	h.setRefreshCookie(c, result.Auth.RefreshToken)
	response.Success(c, gin.H{
		"access_token": result.Auth.AccessToken,
		"user":         result.Auth.User,
	})
}

// VerifyTOTP godoc
// @Summary      Verify TOTP code after login
// @Description  Completes the 2FA login step. Sets refresh_token cookie and returns access_token + user.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/totp-verify [post]
func (h *AuthHandler) VerifyTOTP(c *gin.Context) {
	var req struct {
		ChallengeToken string `json:"challenge_token" binding:"required"`
		Code           string `json:"code"            binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.VerifyTOTP(req.ChallengeToken, req.Code, c.GetHeader("User-Agent"), c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}

	h.setRefreshCookie(c, resp.RefreshToken)
	response.Success(c, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
	})
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Issues a new access token by reading the refresh_token httpOnly cookie. No request body required.
// @Tags         auth
// @Produce      json
// @Success      200   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	cookie, err := c.Cookie(refreshTokenCookie)
	if err != nil {
		response.Error(c, apperror.Unauthorized("no refresh token"))
		return
	}

	resp, err := h.svc.Refresh(cookie)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Logout godoc
// @Summary      Logout current session
// @Description  Invalidates the session identified by the refresh_token cookie and clears it.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, err := c.Cookie(refreshTokenCookie)
	if err == nil {
		if svcErr := h.svc.Logout(cookie); svcErr != nil {
			response.Error(c, svcErr)
			return
		}
	}
	h.clearRefreshCookie(c)
	response.Success(c, gin.H{"message": "logged out"})
}

// LogoutAll godoc
// @Summary      Logout all sessions
// @Description  Invalidates all refresh tokens for the authenticated user
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	if err := h.svc.LogoutAll(userID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "logged out from all devices"})
}

// Me godoc
// @Summary      Get current user profile
// @Description  Returns the authenticated user's profile. Used by the frontend to rehydrate user state.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	profile, err := h.svc.GetProfile(userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":           profile.ID,
			"email":        profile.Email,
			"name":         profile.Name,
			"avatar_url":   profile.AvatarURL,
			"provider":     profile.Provider,
			"totp_enabled": profile.TOTPEnabled,
			"created_at":   profile.CreatedAt,
		},
	})
}
