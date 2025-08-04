// api/user_handler.go
package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/pranav244872/synapse/db/sqlc"
)

// userProfileResponse defines the structure for the /users/me endpoint response.
type userProfileResponse struct {
	Name  string       `json:"name"`
	Email string       `json:"email"`
	Role  db.UserRole `json:"role"`
}

// getUserProfile handles the GET /users/me endpoint.
// It uses the user ID from the JWT payload to fetch the user's profile.
func (server *Server) getUserProfile(ctx *gin.Context) {
	// 1. Get the payload from the context (set by the authMiddleware).
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// 2. Extract the user ID from the payload.
	// The user ID is stored as a float64 in JWT claims, so we need to cast it.
	userID := int64(authPayload["user_id"].(float64))

	// 3. Fetch the user's data from the database using their ID.
	user, err := server.store.GetUser(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// This could happen if the user was deleted after the token was issued.
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 4. Create the response object with the required fields.
	rsp := userProfileResponse{
		Name:  user.Name.String, // pgtype.Text needs to be converted to string
		Email: user.Email,
		Role:  user.Role,
	}

	// 5. Send the response.
	ctx.JSON(http.StatusOK, rsp)
}
