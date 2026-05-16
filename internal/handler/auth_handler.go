package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/internal/service"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/response"
)

type AuthHandler struct {
	svc service.AuthServiceInterface
}

func NewAuthHandler(svc service.AuthServiceInterface) *AuthHandler {
	return &AuthHandler{svc: svc}
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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a local user account and returns an access/refresh token pair
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
	response.Created(c, resp)
}

// Login godoc
// @Summary      Login
// @Description  Verifies credentials and returns an access/refresh token pair
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

	resp, err := h.svc.Login(service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Issues a new access token using a valid refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      refreshRequest  true  "Refresh token"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	resp, err := h.svc.Refresh(req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Logout godoc
// @Summary      Logout current session
// @Description  Invalidates the given refresh token (deletes the session)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      logoutRequest  true  "Refresh token to invalidate"
// @Success      200   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	if err := h.svc.Logout(req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
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
			"id":         profile.ID,
			"email":      profile.Email,
			"name":       profile.Name,
			"avatar_url": profile.AvatarURL,
			"provider":   profile.Provider,
			"created_at": profile.CreatedAt,
		},
	})
}
