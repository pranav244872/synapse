package api

import (
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
	"github.com/pranav244872/synapse/util"

	"github.com/gin-gonic/gin"
)

////////////////////////////////////////////////////////////////////////
// Protected Endpoint: Create Invitation
////////////////////////////////////////////////////////////////////////

// Request body structure for creating an invitation
type createInvitationRequest struct {
	Email string       `json:"email" binding:"required,email"`               // Email to invite
	Role  db.UserRole  `json:"role" binding:"required,oneof=manager engineer"` // Role of the user being invited
}

// createInvitation allows a logged-in user (e.g., manager) to invite someone.
// It uses JWT middleware to authenticate and authorize the inviter.
func (server *Server) createInvitation(ctx *gin.Context) {
	var req createInvitationRequest

	// Bind and validate incoming JSON request
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Extract user claims (user ID, role) from JWT token
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Convert user_id from float64 to int64 (Go quirk)
	inviterID := int64(authPayload["user_id"].(float64))

	// Build DB parameters for transactional invitation creation
	arg := db.CreateInvitationTxParams{
		InviterID:     inviterID,
		EmailToInvite: req.Email,
		RoleToInvite:  req.Role,
	}

	// Run the transaction to create the invitation
	result, err := server.store.CreateInvitationTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Return the created invitation details
	ctx.JSON(http.StatusOK, result.Invitation)
}

////////////////////////////////////////////////////////////////////////
// Public Endpoint: Accept Invitation & Register
////////////////////////////////////////////////////////////////////////

// Request body for accepting an invitation and onboarding
type acceptInvitationRequest struct {
	Token      string `json:"token" binding:"required"`             // Token from the invite email
	Name       string `json:"name" binding:"required"`              // Name of the user
	Password   string `json:"password" binding:"required,min=6"`    // Chosen password
	ResumeText string `json:"resumeText" binding:"required"`        // Resume content for skill extraction
}

// Response structure after successful invitation acceptance
type acceptInvitationResponse struct {
	User  db.User `json:"user"`  // Created user info
	Token string  `json:"token"` // JWT token for authentication
}

// acceptInvitation processes a token and creates a new user with extracted skills
func (server *Server) acceptInvitation(ctx *gin.Context) {
	var req acceptInvitationRequest

	// Validate and parse request body
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// 1. Validate invitation token
	invitation, err := server.store.GetInvitationByToken(ctx, req.Token)
	if err != nil {
		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err)) // Token not found
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Check if invitation is still valid and not expired
	if invitation.Status != "pending" || time.Now().After(invitation.ExpiresAt.Time) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invitation is invalid or has expired"})
		return
	}

	// 2. Hash password securely
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 3. Extract skills from resume using skillzProcessor
	// Normalize skill names from resume
	normalizedSkills, err := server.skillzProcessor.ExtractAndNormalize(ctx, req.ResumeText)
	if err != nil {
		log.Printf("❌ skillzProcessor error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not process resume skills"})
		return
	}

	// Try to extract proficiency levels per skill (optional)
	proficiencies, err := server.skillzProcessor.ExtractProficiencies(ctx, req.ResumeText, normalizedSkills)
	if err != nil {
		// Proceed anyway — not critical to fail here
		log.Printf("⚠️ Could not extract proficiencies for user %s: %v\n", req.Name, err)
	}

	// Convert proficiencies to the format expected by the database
	skillsWithProficiency := make(map[string]db.ProficiencyLevel)
	for skill, prof := range proficiencies {
		skillsWithProficiency[skill] = db.ProficiencyLevel(prof)
	}

	// 4. Onboard new user with skills and hashed password
	onboardParams := db.OnboardNewUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Name:          pgtype.Text{String: req.Name, Valid: true},
			Email:         invitation.Email,
			PasswordHash:  hashedPassword,
			Role:          invitation.RoleToInvite,
		},
		SkillsWithProficiency: skillsWithProficiency,
	}

	onboardResult, err := server.store.OnboardNewUserWithSkills(ctx, onboardParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 5. Mark invitation as accepted (non-critical step)
	_, err = server.store.UpdateInvitationStatus(ctx, db.UpdateInvitationStatusParams{
		ID:     invitation.ID,
		Status: "accepted",
	})
	if err != nil {
		// Not critical — user is already created
		log.Printf("⚠️ Failed to update invitation status for user %d: %v\n", onboardResult.User.ID, err)
	}

	// 6. Create a JWT token for the newly onboarded user
	token, err := server.tokenMaker.CreateToken(
		onboardResult.User.ID,
		onboardResult.User.Role,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// 7. Return response with user and token
	rsp := acceptInvitationResponse{
		User:  onboardResult.User,
		Token: token,
	}
	ctx.JSON(http.StatusOK, rsp)
}
