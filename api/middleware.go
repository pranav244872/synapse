package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pranav244872/synapse/token"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

////////////////////////////////////////////////////////////////////////
// Constants used in authMiddleware
////////////////////////////////////////////////////////////////////////

const (
	authorizationHeaderKey  = "authorization"             // HTTP header where token is expected
	authorizationTypeBearer = "bearer"                    // Authorization type: Bearer <token>
	authorizationPayloadKey = "authorization_payload"     // Context key for storing the token payload
)

////////////////////////////////////////////////////////////////////////
// Middleware to authenticate JWTs
////////////////////////////////////////////////////////////////////////

// authMiddleware checks for a valid JWT token in the "Authorization" header.
// If valid, it stores the decoded token (claims) in Gin's context for use in handlers.
// If invalid or missing, it blocks access with a 401 Unauthorized.
func authMiddleware(tokenMaker *token.JWTMaker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1. Get the value of the Authorization header
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		// If the header is missing, reject the request
		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// 2. The expected format is: "Bearer <token>"
		// We split the header into parts: ["Bearer", "<token>"]
		fields := strings.Fields(authorizationHeader)

		// If the header is not in the right format (missing token), reject
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// 3. Check that the type is "Bearer" (case-insensitive)
		authType := strings.ToLower(fields[0])
		if authType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// 4. Extract the actual token string
		accessToken := fields[1]

		// 5. Validate the JWT token
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			// If the token is invalid (expired, forged, etc.), reject
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// 6. If valid, save the token claims (payload) to the Gin context
		// This makes the user data available to future handlers
		ctx.Set(authorizationPayloadKey, payload)

		// 7. Continue to the next handler (e.g. createInvitation)
		ctx.Next()
	}
}

////////////////////////////////////////////////////////////////////////
// Helper to extract JWT claims from context
////////////////////////////////////////////////////////////////////////

// getAuthorizationPayload is a helper function that lets your handlers
// retrieve the JWT claims (user ID, role, etc.) that the middleware stored.
// Example use: getAuthorizationPayload(ctx) in createInvitation()
func getAuthorizationPayload(ctx *gin.Context) (jwt.MapClaims, error) {
	// 1. Get the stored payload from context using the key "authorization_payload"
	payload, exists := ctx.Get(authorizationPayloadKey)
	if !exists {
		// If not found, probably the middleware was not run
		return nil, errors.New("authorization payload not found")
	}

	// 2. Check that the payload is of the correct type (jwt.MapClaims)
	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		// Type mismatch - maybe wrong value stored
		return nil, errors.New("invalid authorization payload type")
	}

	// 3. Return the claims (which is just a map of keys like "user_id", "role")
	return claims, nil
}
