package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/internal/service"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/response"
)

type ProfileHandler struct {
	svc service.ProfileServiceInterface
}

func NewProfileHandler(svc service.ProfileServiceInterface) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

type updateProfileRequest struct {
	Name   string `json:"name"   binding:"required"`
	Email  string `json:"email"  binding:"required,email"`
	Locale string `json:"locale" binding:"omitempty"`
}

type deleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}

type profileResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Provider  string `json:"provider"`
	Locale    string `json:"locale"`
}

// Update godoc
// @Summary      Update user profile
// @Description  Updates the authenticated user's name and email
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        body  body      updateProfileRequest  true  "Profile fields"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      409   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /profile [patch]
func (h *ProfileHandler) Update(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextUserID)
	info, err := h.svc.UpdateProfile(userID, service.UpdateProfileRequest{
		Name:   req.Name,
		Email:  req.Email,
		Locale: req.Locale,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, profileResponse{
		ID:        info.ID,
		Name:      info.Name,
		Email:     info.Email,
		AvatarURL: info.AvatarURL,
		Provider:  info.Provider,
		Locale:    info.Locale,
	})
}

// DeleteAccount godoc
// @Summary      Delete account
// @Description  Verifies password and permanently deletes the authenticated user and all their data
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        body  body      deleteAccountRequest  true  "Current password"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /profile [delete]
func (h *ProfileHandler) DeleteAccount(c *gin.Context) {
	var req deleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextUserID)
	if err := h.svc.DeleteAccount(userID, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// UploadAvatar godoc
// @Summary      Upload profile avatar
// @Description  Uploads an image to Cloudflare R2 and sets it as the user's avatar
// @Tags         profile
// @Accept       multipart/form-data
// @Produce      json
// @Param        avatar  formData  file  true  "Avatar image (JPEG/PNG/WebP, max 5 MB)"
// @Success      200     {object}  map[string]interface{}
// @Failure      400     {object}  map[string]interface{}
// @Failure      401     {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /profile/avatar [post]
func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	const maxMemory = 5 << 20 // 5 MB
	if err := c.Request.ParseMultipartForm(maxMemory); err != nil {
		response.Error(c, apperror.BadRequest("failed to parse form: "+err.Error()))
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Error(c, apperror.BadRequest("avatar field is required"))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	userID := c.GetString(middleware.ContextUserID)
	url, err := h.svc.UploadAvatar(userID, header.Filename, contentType, header.Size, file)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"avatar_url": url})
}
