package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/response"
)

type forgotPasswordRequestBody struct {
	Email string `json:"email" binding:"required,email"`
}

type forgotPasswordResetBody struct {
	Email       string `json:"email"        binding:"required,email"`
	OTP         string `json:"otp"          binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ForgotPasswordRequest godoc
// @Summary      Request password reset code
// @Description  Sends a 6-digit OTP to the given email if it is registered. Always returns 200 to prevent email enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      forgotPasswordRequestBody  true  "Email"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /auth/forgot-password/request [post]
func (h *SecurityHandler) ForgotPasswordRequest(c *gin.Context) {
	var req forgotPasswordRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.RequestPasswordReset(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "If that email is registered, a code has been sent."})
}

// ForgotPasswordReset godoc
// @Summary      Reset password with OTP
// @Description  Verifies the OTP sent to the given email and updates the password. Invalidates all active sessions.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      forgotPasswordResetBody  true  "Reset payload"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Router       /auth/forgot-password/reset [post]
func (h *SecurityHandler) ForgotPasswordReset(c *gin.Context) {
	var req forgotPasswordResetBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Email, req.OTP, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Password reset successfully."})
}
