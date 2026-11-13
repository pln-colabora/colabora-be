package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/auth/service"
	"github.com/stretchr/testify/require"
)

func TestAuthenticateRejectsPasswordResetToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtService := service.NewJWTService()
	server := gin.New()
	server.GET("/me", Authenticate(jwtService), func(ctx *gin.Context) { ctx.Status(http.StatusOK) })

	send := func(token string) int {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/me", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		server.ServeHTTP(recorder, request)
		return recorder.Code
	}

	require.Equal(t, http.StatusOK, send(jwtService.GenerateAccessToken("user-id", "teknik")))
	require.Equal(t, http.StatusUnauthorized, send(jwtService.GenerateAccessToken("user-id", service.PasswordResetTokenRole)))
}
