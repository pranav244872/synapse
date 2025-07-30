package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/pranav244872/synapse/util"
)

////////////////////////////////////////////////////////////////////////
// Login Endpoint (Public): /auth/login
////////////////////////////////////////////////////////////////////////

// loginUserRequest defines the expected JSON payload for login.
// Example:
// {
//   "email": "user@example.com",
//   "password": "securepassword"
// }
type loginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`     // Required field, must be a valid email
	Password string `json:"password" binding:"required,min=6"`  // Required field, minimum 6 characters
}

// loginUserResponse defines the structure of a successful login response.
// It contains a signed JWT token the client can use for authenticated requests.
type loginUserResponse struct {
	Token string `json:"token"` // Access token for subsequent requests
}

////////////////////////////////////////////////////////////////////////
// Handler: loginUser
// Authenticates a user using email and password.
// Returns a signed JWT token if credentials are valid.
////////////////////////////////////////////////////////////////////////

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest

	// Step 1: Bind and validate the request body (email and password)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// If JSON is malformed or fields are invalid, respond with 400
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Step 2: Retrieve user from the database by email
	user, err := server.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		// If no user is found with that email, respond with 404
		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		// For other database errors, respond with 500
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Step 3: Check that the provided password matches the stored hash
	err = util.CheckPasswordHash(req.Password, user.PasswordHash)
	if err != nil {
		// If the password is incorrect, respond with 401 Unauthorized
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Step 4: Generate a JWT token for the authenticated user
	token, err := server.tokenMaker.CreateToken(
		user.ID,                      // Include user ID in the token payload
		user.Role,                    // Include user role (e.g. engineer, manager)
		server.config.AccessTokenDuration, // Token expiration (from config)
	)
	if err != nil {
		// Token generation failure (should rarely happen)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Step 5: Send response with token
	rsp := loginUserResponse{
		Token: token,
	}

	// Return 200 OK with the token so the client can store and use it
	ctx.JSON(http.StatusOK, rsp)
}
