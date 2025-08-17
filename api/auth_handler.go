// api/auth_handler.go

package api

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
	"github.com/pranav244872/synapse/util"
)

////////////////////////////////////////////////////////////////////////
// Login Endpoint (Public): /auth/login
////////////////////////////////////////////////////////////////////////

// loginUserRequest defines the expected JSON payload for login.
// Example:
//
//	{
//	  "email": "user@example.com",
//	  "password": "securepassword"
//	}
type loginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`    // Required field, must be a valid email
	Password string `json:"password" binding:"required,min=6"` // Required field, minimum 6 characters
}

// loginUserResponse defines the structure of a successful login response.
// It contains a signed JWT token the client can use for authenticated requests.
type loginUserResponse struct {
	Token string `json:"token"` // Access token for subsequent requests
}

////////////////////////////////////////////////////////////////////////
// Handler: loginUser
// Authenticates a user using email and password.
// Returns a signed JWT token with user_id, role and team_id if credentials are valid.
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
		user.ID,                           // Include user ID in the token payload
		user.Role,                         // Include user role (e.g. engineer, manager)
		user.TeamID,                       // Pass the user's team Id to the token manker
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

////////////////////////////////////////////////////////////////////////
// Accept Invitation Endpoint (Public)
////////////////////////////////////////////////////////////////////////

// acceptInvitationRequest defines the JSON body for the accept invitation endpoint.
type acceptInvitationRequest struct {
	Token      string `json:"token" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Password   string `json:"password" binding:"required,min=6"`
	ResumeText string `json:"resume_text" binding:"required"`
}

// userResponse is a cleaner struct for API output, omitting the password hash.
type userResponse struct {
	ID     int64       `json:"id"`
	Name   string      `json:"name"`
	Email  string      `json:"email"`
	Role   db.UserRole `json:"role"`
	TeamID pgtype.Int8 `json:"team_id"`
}

// acceptInvitationResponse defines the successful response structure.
type acceptInvitationResponse struct {
	User  userResponse `json:"user"`
	Token string       `json:"token"`
}

func (server *Server) acceptInvitation(ctx *gin.Context) {
	var req acceptInvitationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	skills, err := server.skillzProcessor.ExtractAndNormalize(ctx, req.ResumeText)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not process resume skills"})
		return
	}
	skillsWithProficiency := make(map[string]db.ProficiencyLevel)
	for _, skillName := range skills {
		skillsWithProficiency[skillName] = db.ProficiencyLevelBeginner
	}

	// Prepare parameters for the NEW, correct transaction.
	txParams := db.AcceptInvitationTxParams{
		InvitationToken:       req.Token,
		UserName:              req.Name,
		PasswordHash:          hashedPassword,
		SkillsWithProficiency: skillsWithProficiency,
	}

	// Execute the new transaction.
	result, err := server.store.AcceptInvitationTx(ctx, txParams)
	if err != nil {
		if errors.Is(err, db.ErrInvitationNotPending) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// After the user is Successfully created call this to update recommender service of the new employee
	server.notifyRecommender()

	// Generate a session JWT for the newly created user.
	jwtToken, err := server.tokenMaker.CreateToken(
		result.User.ID,
		result.User.Role,
		result.User.TeamID,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Format and send the final response.
	rsp := acceptInvitationResponse{
		User: userResponse{
			ID:     result.User.ID,
			Name:   result.User.Name.String,
			Email:  result.User.Email,
			Role:   result.User.Role,
			TeamID: result.User.TeamID,
		},
		Token: jwtToken,
	}

	ctx.JSON(http.StatusOK, rsp)
}

////////////////////////////////////////////////////////////////////////
// Helper functions
////////////////////////////////////////////////////////////////////////

// notifyRecommender sends a non-blocking POST request to the recommender service
// to trigger a model refresh. It runs in a separate goroutine.
func (server *Server) notifyRecommender() {
	// Fire-and-forget: run this in the background so it doesn't block the API response.
	go func() {
		recommenderBaseURL := server.config.RecommenderAPIURL
		recommenderAPIKey := server.config.RecommenderAPIKey

		if recommenderBaseURL == "" || recommenderAPIKey == "" {
			log.Println("WARN: Recommender service URL or API key is not configured. Skipping notification.")
			return
		}

		// Safely parse the base URL.
		parsedURL, err := url.Parse(recommenderBaseURL)
		if err != nil {
			log.Printf("ERROR: Failed to parse recommender base URL '%s': %v", recommenderBaseURL, err)
			return
		}

		// Safely join the path to the base URL.
		parsedURL.Path = path.Join(parsedURL.Path, "/admin/refresh-model")
		endpointURL := parsedURL.String()

		log.Printf("INFO: Notifying recommender service at: %s", endpointURL)

		// Create a new HTTP client with a reasonable timeout.
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		// Create the POST request with an empty body.
		req, err := http.NewRequest("POST", endpointURL, nil)
		if err != nil {
			log.Printf("ERROR: Failed to create request for recommender service: %v", err)
			return
		}

		// Set the required API key header for authentication.
		req.Header.Set("X-Internal-API-Key", recommenderAPIKey)

		// Send the request.
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ERROR: Failed to send request to recommender service: %v", err)
			return
		}
		defer resp.Body.Close()

		// Check the response status. The recommender should return 202 Accepted.
		if resp.StatusCode != http.StatusAccepted {
			log.Printf("ERROR: Recommender service returned a non-202 status: %d", resp.StatusCode)
			return
		}

		log.Println("INFO: Successfully notified recommender service to refresh its model.")
	}()
}
