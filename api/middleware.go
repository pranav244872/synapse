// api/middleware.go

package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	db "github.com/pranav244872/synapse/db/sqlc"
	"github.com/pranav244872/synapse/token"
)

// Constants used for auth
const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

////////////////////////////////////////////////////////////////////////
// CORS MIDDLEWARE
////////////////////////////////////////////////////////////////////////

// CORSMiddleware creates a gin.HandlerFunc that sets the required CORS headers.
// It reads the allowed origin from the server's configuration.
func (server *Server) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", server.config.FrontendURL)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

////////////////////////////////////////////////////////////////////////
// AUTHENTICATION MIDDLEWARE
////////////////////////////////////////////////////////////////////////

// authMiddleware checks for a valid JWT and stores its payload in the context.
func authMiddleware(tokenMaker *token.JWTMaker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authType := strings.ToLower(fields[0])
		if authType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

////////////////////////////////////////////////////////////////////////
// AUTHORIZATION MIDDLEWARE (ROLE-BASED)
////////////////////////////////////////////////////////////////////////

// adminAuthMiddleware checks if the user has the 'admin' role.
// It must be used AFTER authMiddleware.
func adminAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get the payload that authMiddleware stored in the context.
		payload, err := getAuthorizationPayload(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// Check the 'role' claim from the token.
		if payload["role"] != string(db.UserRoleAdmin) {
			err := errors.New("this action requires admin privileges")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err)) // 403 Forbidden
			return
		}

		ctx.Next()
	}
}

// managerAuthMiddleware checks if the user has the 'manager' role.
// It must be used AFTER authMiddleware.
func managerAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		payload, err := getAuthorizationPayload(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		if payload["role"] != string(db.UserRoleManager) {
			err := errors.New("this action requires manager privileges")
			ctx.AbortWithStatusJSON(http.StatusForbidden, errorResponse(err)) // 403 Forbidden
			return
		}

		ctx.Next()
	}
}

////////////////////////////////////////////////////////////////////////
// HELPER FUNCTION
////////////////////////////////////////////////////////////////////////

// getAuthorizationPayload retrieves the JWT claims from the context.
func getAuthorizationPayload(ctx *gin.Context) (jwt.MapClaims, error) {
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		return nil, errors.New("authorization payload not found")
	}

	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid authorization payload type")
	}

	return claims, nil
}
