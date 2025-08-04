// api/admin_handler.go
package api

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
)

// Generic type in Go for paginated responses using Go 1.18+ generics.
// This is useful when returning lists of any type with metadata.
type paginatedResponse[T any] struct {
	TotalCount int64 `json:"total_count"` // Total items matching the query, useful for frontend pagination.
	Data       []T   `json:"data"`        // The current page of results.
}

////////////////////////////////////////////////////////////////////////
// Teams Management
////////////////////////////////////////////////////////////////////////

type listTeamsRequest struct {
	PageID    int32 `form:"page_id" binding:"omitempty,required_without=Unmanaged,min=1"`
	PageSize  int32 `form:"page_size" binding:"omitempty,required_without=Unmanaged,min=5,max=20"`
	Unmanaged *bool `form:"unmanaged"`
}

// listTeams handles retrieving teams with proper pagination and filtering
func (server *Server) listTeams(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listTeams handler")

	var req listTeamsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		log.Printf("DEBUG: Teams query bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Teams request params - PageID: %d, PageSize: %d, Unmanaged: %v", 
		req.PageID, req.PageSize, req.Unmanaged)

	// This branch is optimized for dropdowns or selection lists in UIs
	if req.Unmanaged != nil && *req.Unmanaged {
		log.Printf("DEBUG: Processing unmanaged teams request")
		unmanagedTeams, err := server.store.ListUnmanagedTeams(ctx)
		if err != nil {
			log.Printf("DEBUG: Error listing unmanaged teams: %v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
		log.Printf("DEBUG: Successfully retrieved %d unmanaged teams", len(unmanagedTeams))
		ctx.JSON(http.StatusOK, unmanagedTeams)
		return
	}

	// Offset = (page - 1) * size is standard pagination logic
	arg := db.ListTeamsWithManagersParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	log.Printf("DEBUG: Querying teams with limit: %d, offset: %d", arg.Limit, arg.Offset)

	teams, err := server.store.ListTeamsWithManagers(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error listing teams with managers: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	totalCount, err := server.store.CountTeams(ctx) // Needed for pagination metadata
	if err != nil {
		log.Printf("DEBUG: Error counting teams: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully retrieved %d teams, total count: %d", len(teams), totalCount)

	rsp := paginatedResponse[db.ListTeamsWithManagersRow]{
		TotalCount: totalCount,
		Data:       teams,
	}

	ctx.JSON(http.StatusOK, rsp)
}

type createTeamRequest struct {
	TeamName string `json:"team_name" binding:"required"` // JSON body binding using struct tags
}

// createTeamAdmin handles creating a new team by admin users
func (server *Server) createTeamAdmin(ctx *gin.Context) {
	log.Printf("DEBUG: Starting createTeamAdmin handler")

	var req createTeamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DEBUG: Create team JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Creating team with name: '%s'", req.TeamName)

	arg := db.CreateTeamParams{
		TeamName: req.TeamName,
	}

	team, err := server.store.CreateTeam(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error creating team: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully created team with ID: %d", team.ID)
	ctx.JSON(http.StatusCreated, team)
}

////////////////////////////////////////////////////////////////////////
// Invitations Management
////////////////////////////////////////////////////////////////////////

type listAdminInvitationsRequest struct {
	PageID      int32  `form:"page_id" binding:"required,min=1"`
	PageSize    int32  `form:"page_size" binding:"required,min=5,max=20"`
	InviterID   string `form:"inviter_id"`
	InviterRole string `form:"inviter_role" binding:"omitempty,oneof=admin manager"`
}

type invitationResponse struct {
	ID           int64            `json:"id"`
	Email        string           `json:"email"`
	RoleToInvite db.UserRole      `json:"role_to_invite"`
	Status       string           `json:"status"`
	InviterName  string           `json:"inviter_name"`
	InviterRole  string           `json:"inviter_role"`
	CreatedAt    pgtype.Timestamp `json:"created_at"`
}

// listInvitations handles retrieving invitations with filtering and pagination
func (server *Server) listInvitations(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listInvitations handler")

	var req listAdminInvitationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		log.Printf("DEBUG: Invitations query bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Invitations request params - PageID: %d, PageSize: %d, InviterID: '%s', InviterRole: '%s'", 
		req.PageID, req.PageSize, req.InviterID, req.InviterRole)

	var finalInvitations []invitationResponse
	var totalCount int64
	var err error

	// Helper function to convert different SQLC row types to our unified response type
	toResponse := func(i any) invitationResponse {
		switch v := i.(type) {
		case db.ListAllInvitationsRow:
			// Handle the interface{} type for InviterRole in ListAllInvitationsRow
			inviterRole := "unknown"
			if role, ok := v.InviterRole.(string); ok {
				inviterRole = role
			}
			log.Printf("DEBUG: Converting ListAllInvitationsRow - ID: %d, InviterRole: %s", v.ID, inviterRole)
			return invitationResponse{
				ID: v.ID, Email: v.Email, RoleToInvite: v.RoleToInvite, Status: v.Status,
				InviterName: v.InviterName, InviterRole: inviterRole, CreatedAt: v.CreatedAt,
			}
		case db.ListInvitationsByInviterRow:
			// InviterRole is already string type for this struct
			log.Printf("DEBUG: Converting ListInvitationsByInviterRow - ID: %d, InviterRole: %s", v.ID, v.InviterRole)
			return invitationResponse{
				ID: v.ID, Email: v.Email, RoleToInvite: v.RoleToInvite, Status: v.Status,
				InviterName: v.InviterName, InviterRole: v.InviterRole, CreatedAt: v.CreatedAt,
			}
		case db.ListInvitationsByInviterRoleRow:
			// InviterRole is already string type for this struct
			log.Printf("DEBUG: Converting ListInvitationsByInviterRoleRow - ID: %d, InviterRole: %s", v.ID, v.InviterRole)
			return invitationResponse{
				ID: v.ID, Email: v.Email, RoleToInvite: v.RoleToInvite, Status: v.Status,
				InviterName: v.InviterName, InviterRole: v.InviterRole, CreatedAt: v.CreatedAt,
			}
		default:
			log.Printf("DEBUG: Unknown invitation type: %T", v)
			return invitationResponse{}
		}
	}

	// Route to appropriate query based on request parameters
	switch {
	case req.InviterID == "me":
		log.Printf("DEBUG: Processing 'me' case - getting current user's invitations")

		// Get authorization payload with proper error handling
		authPayload, err := getAuthorizationPayload(ctx)
		if err != nil {
			log.Printf("DEBUG: Failed to get authorization payload: %v", err)
			ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
			return
		}

		// Safely extract user_id with type assertion
		userIDFloat, ok := authPayload["user_id"].(float64)
		if !ok {
			log.Printf("DEBUG: user_id not found or not a float64 in auth payload. Payload: %+v", authPayload)
			ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("invalid user_id in token")))
			return
		}

		adminID := int64(userIDFloat)
		log.Printf("DEBUG: Extracted Admin ID: %d", adminID)

		// Query invitations by specific inviter
		invitations, dbErr := server.store.ListInvitationsByInviter(ctx, db.ListInvitationsByInviterParams{
			InviterID: adminID,
			Limit:     req.PageSize,
			Offset:    (req.PageID - 1) * req.PageSize,
		})
		err = dbErr
		if err == nil {
			log.Printf("DEBUG: Retrieved %d invitations by inviter", len(invitations))
			totalCount, err = server.store.CountInvitationsByInviter(ctx, adminID)
			if err != nil {
				log.Printf("DEBUG: Error counting invitations by inviter: %v", err)
			} else {
				log.Printf("DEBUG: Total count by inviter: %d", totalCount)
			}
			// Convert each invitation to response format
			for _, inv := range invitations {
				finalInvitations = append(finalInvitations, toResponse(inv))
			}
		} else {
			log.Printf("DEBUG: Error listing invitations by inviter: %v", err)
		}

	case req.InviterRole != "":
		log.Printf("DEBUG: Processing inviter role case: %s", req.InviterRole)

		// Query invitations by inviter role
		invitations, dbErr := server.store.ListInvitationsByInviterRole(ctx, db.ListInvitationsByInviterRoleParams{
			Role:   db.UserRole(req.InviterRole),
			Limit:  req.PageSize,
			Offset: (req.PageID - 1) * req.PageSize,
		})
		err = dbErr
		if err == nil {
			log.Printf("DEBUG: Retrieved %d invitations by role", len(invitations))
			totalCount, err = server.store.CountInvitationsByInviterRole(ctx, db.UserRole(req.InviterRole))
			if err != nil {
				log.Printf("DEBUG: Error counting invitations by role: %v", err)
			} else {
				log.Printf("DEBUG: Total count by role: %d", totalCount)
			}
			// Convert each invitation to response format
			for _, inv := range invitations {
				finalInvitations = append(finalInvitations, toResponse(inv))
			}
		} else {
			log.Printf("DEBUG: Error listing invitations by role: %v", err)
		}

	default:
		log.Printf("DEBUG: Processing default case (all invitations)")

		// Query all invitations
		invitations, dbErr := server.store.ListAllInvitations(ctx, db.ListAllInvitationsParams{
			Limit:  req.PageSize,
			Offset: (req.PageID - 1) * req.PageSize,
		})
		err = dbErr
		if err == nil {
			log.Printf("DEBUG: Retrieved %d all invitations", len(invitations))
			totalCount, err = server.store.CountAllInvitations(ctx)
			if err != nil {
				log.Printf("DEBUG: Error counting all invitations: %v", err)
			} else {
				log.Printf("DEBUG: Total count all: %d", totalCount)
			}
			// Convert each invitation to response format
			for _, inv := range invitations {
				finalInvitations = append(finalInvitations, toResponse(inv))
			}
		} else {
			log.Printf("DEBUG: Error listing all invitations: %v", err)
		}
	}

	// Handle any errors that occurred during database operations
	if err != nil {
		log.Printf("DEBUG: Final error before returning 500: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully processed, returning %d invitations", len(finalInvitations))

	// Build paginated response
	rsp := paginatedResponse[invitationResponse]{
		TotalCount: totalCount,
		Data:       finalInvitations,
	}
	ctx.JSON(http.StatusOK, rsp)
}

type createManagerInvitationRequest struct {
	Email  string `json:"email" binding:"required,email"`
	TeamID int64  `json:"team_id" binding:"required,min=1"`
}

// createManagerInvitation handles creating invitations for manager role
func (server *Server) createManagerInvitation(ctx *gin.Context) {
	log.Printf("DEBUG: Starting createManagerInvitation handler")

	var req createManagerInvitationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DEBUG: Create manager invitation JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Creating manager invitation - Email: %s, TeamID: %d", req.Email, req.TeamID)

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for invitation creation: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract user_id with type assertion
	userIDFloat, ok := authPayload["user_id"].(float64)
	if !ok {
		log.Printf("DEBUG: user_id not found or not a float64 in auth payload for invitation creation. Payload: %+v", authPayload)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("invalid user_id in token")))
		return
	}

	inviterID := int64(userIDFloat)
	log.Printf("DEBUG: Extracted Inviter ID: %d", inviterID)

	// Use the new CreateInvitationTx transaction function instead of the basic CreateInvitation
	arg := db.CreateInvitationTxParams{
		InviterID:     inviterID,
		EmailToInvite: req.Email,
		RoleToInvite:  db.UserRoleManager,
		TeamID:        pgtype.Int8{Int64: req.TeamID, Valid: true},
	}

	log.Printf("DEBUG: Calling CreateInvitationTx with params: %+v", arg)

	result, err := server.store.CreateInvitationTx(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error creating invitation: %v", err)

		// Handle specific business logic errors from the transaction
		switch {
		case errors.Is(err, db.ErrPermissionDenied):
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		case errors.Is(err, db.ErrDuplicateInvitation):
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		case errors.Is(err, db.ErrInvalidRoleSequence):
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		case errors.Is(err, db.ErrTeamIDRequiredForManager):
			ctx.JSON(http.StatusBadRequest, errorResponse(err))
			return
		case errors.Is(err, db.ErrTeamNotFound):
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		case errors.Is(err, db.ErrTeamAlreadyHasManager):
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		default:
			// Generic database or system error
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	log.Printf("DEBUG: Successfully created invitation with ID: %d, Token: %s, Expires: %v", 
		result.Invitation.ID, result.Invitation.InvitationToken, result.Invitation.ExpiresAt.Time)

	// Return the created invitation details
	ctx.JSON(http.StatusCreated, result.Invitation)
}

type deleteInvitationRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// deleteInvitation handles removing pending invitations
func (server *Server) deleteInvitation(ctx *gin.Context) {
	log.Printf("DEBUG: Starting deleteInvitation handler")

	var req deleteInvitationRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		log.Printf("DEBUG: Delete invitation URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Deleting invitation with ID: %d", req.ID)

	// First, check if the invitation exists and get its status
	invitation, err := server.store.GetInvitationByID(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("DEBUG: Invitation not found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("invitation not found")))
			return
		}
		log.Printf("DEBUG: Error checking invitation: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Check if invitation can be deleted
	if invitation.Status != "pending" {
		log.Printf("DEBUG: Cannot delete invitation with status: %s", invitation.Status)
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("only pending invitations can be deleted")))
		return
	}

	// Proceed with deletion
	err = server.store.DeleteInvitation(ctx, req.ID)
	if err != nil {
		log.Printf("DEBUG: Error deleting invitation: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully deleted invitation with ID: %d", req.ID)
	ctx.Status(http.StatusNoContent)
}

////////////////////////////////////////////////////////////////////////
// Skills Management
////////////////////////////////////////////////////////////////////////

type listSkillsAdminRequest struct {
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=50"`
	Verified *bool  `form:"verified" binding:"required"`
	Search   string `form:"search"`
}

// listSkillsAdmin handles retrieving skills with verification status filtering
func (server *Server) listSkillsAdmin(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listSkillsAdmin handler")

	var req listSkillsAdminRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		log.Printf("DEBUG: Skills admin query bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Skills admin request params - PageID: %d, PageSize: %d, Verified: %v, Search: '%s'", 
		req.PageID, req.PageSize, *req.Verified, req.Search)

	var skills []db.Skill
	var totalCount int64
	var err error

	// Check if search parameter is provided
	if req.Search != "" {
		// Use search functionality
		searchPattern := "%" + req.Search + "%"

		log.Printf("DEBUG: Searching skills with pattern: %s", searchPattern)

		searchArg := db.SearchSkillsByStatusParams{
			IsVerified: *req.Verified,
			Lower:      searchPattern,
			Limit:      req.PageSize,
			Offset:     (req.PageID - 1) * req.PageSize,
		}

		skills, err = server.store.SearchSkillsByStatus(ctx, searchArg)
		if err != nil {
			log.Printf("DEBUG: Error searching skills by status: %v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		// Get count for search results
		countArg := db.CountSearchSkillsByStatusParams{
			IsVerified: *req.Verified,
			Lower:      searchPattern,
		}

		totalCount, err = server.store.CountSearchSkillsByStatus(ctx, countArg)
		if err != nil {
			log.Printf("DEBUG: Error counting search skills by status: %v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	} else {
		// Use existing functionality without search
		listArg := db.ListSkillsByStatusParams{
			IsVerified: *req.Verified,
			Limit:      req.PageSize,
			Offset:     (req.PageID - 1) * req.PageSize,
		}

		log.Printf("DEBUG: Querying skills with verification status: %v, limit: %d, offset: %d", 
			listArg.IsVerified, listArg.Limit, listArg.Offset)

		skills, err = server.store.ListSkillsByStatus(ctx, listArg)
		if err != nil {
			log.Printf("DEBUG: Error listing skills by status: %v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		totalCount, err = server.store.CountSkillsByStatus(ctx, *req.Verified)
		if err != nil {
			log.Printf("DEBUG: Error counting skills by status: %v", err)
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	log.Printf("DEBUG: Successfully retrieved %d skills, total count: %d", len(skills), totalCount)

	rsp := paginatedResponse[db.Skill]{
		TotalCount: totalCount,
		Data:       skills,
	}
	ctx.JSON(http.StatusOK, rsp)
}

////////////////////////////////////////////////////////////////////////

type updateSkillRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateSkillBody struct {
	IsVerified bool `json:"is_verified"`
}

// updateSkillVerification handles updating skill verification status
func (server *Server) updateSkillVerification(ctx *gin.Context) {
	log.Printf("DEBUG: Starting updateSkillVerification handler")

	var uriReq updateSkillRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		log.Printf("DEBUG: Update skill URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var bodyReq updateSkillBody
	if err := ctx.ShouldBindJSON(&bodyReq); err != nil {
		log.Printf("DEBUG: Update skill JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Updating skill verification - ID: %d, IsVerified: %v", uriReq.ID, bodyReq.IsVerified)

	arg := db.UpdateSkillVerificationParams{
		ID:         uriReq.ID,
		IsVerified: bodyReq.IsVerified,
	}

	skill, err := server.store.UpdateSkillVerification(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error updating skill verification: %v", err)

		if err == sql.ErrNoRows {
			log.Printf("DEBUG: Skill not found for verification update")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("skill not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully updated skill verification for ID: %d", skill.ID)
	ctx.JSON(http.StatusOK, skill)
}

////////////////////////////////////////////////////////////////////////

type deleteSkillRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// deleteSkill handles removing skills from the system
func (server *Server) deleteSkill(ctx *gin.Context) {
	log.Printf("DEBUG: Starting deleteSkill handler")

	var req deleteSkillRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		log.Printf("DEBUG: Delete skill URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Deleting skill with ID: %d", req.ID)

	err := server.store.DeleteSkill(ctx, req.ID)
	if err != nil {
		log.Printf("DEBUG: Error deleting skill: %v", err)

		if err == sql.ErrNoRows {
			log.Printf("DEBUG: Skill not found for deletion")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("skill not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully deleted skill with ID: %d", req.ID)
	ctx.Status(http.StatusNoContent)
}

////////////////////////////////////////////////////////////////////////

type createSkillAliasRequest struct {
	AliasName string `json:"alias_name" binding:"required"`
	SkillID   int64  `json:"skill_id" binding:"required,min=1"`
}

// createSkillAlias handles creating alternative names for skills
func (server *Server) createSkillAlias(ctx *gin.Context) {
	log.Printf("DEBUG: Starting createSkillAlias handler")

	var req createSkillAliasRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DEBUG: Create skill alias JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Creating skill alias - AliasName: %s, SkillID: %d", req.AliasName, req.SkillID)

	// Convert alias name to lowercase for consistency
	normalizedAliasName := strings.ToLower(req.AliasName)
	log.Printf("DEBUG: Normalized alias name: %s", normalizedAliasName)

	arg := db.CreateSkillAliasParams{
		AliasName: normalizedAliasName,
		SkillID:   req.SkillID,
	}

	alias, err := server.store.CreateSkillAlias(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error creating skill alias: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully created skill alias with ID: %d", alias.SkillID)
	ctx.JSON(http.StatusCreated, alias)
}

////////////////////////////////////////////////////////////////////////

// New handler for listing skill aliases
type listSkillAliasesRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// listSkillAliases handles retrieving all aliases for a specific skill
func (server *Server) listSkillAliases(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listSkillAliases handler")

	var req listSkillAliasesRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		log.Printf("DEBUG: List skill aliases URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Listing aliases for skill ID: %d", req.ID)

	// First, verify that the skill exists
	skill, err := server.store.GetSkill(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("DEBUG: Skill not found for aliases listing")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("skill not found")))
			return
		}
		log.Printf("DEBUG: Error checking skill existence: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get all aliases for this skill
	aliases, err := server.store.ListAliasesForSkill(ctx, req.ID)
	if err != nil {
		log.Printf("DEBUG: Error listing aliases for skill: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully retrieved %d aliases for skill '%s'", len(aliases), skill.SkillName)

	// Return both skill info and its aliases
	response := gin.H{
		"skill": skill,
		"aliases": aliases,
	}

	ctx.JSON(http.StatusOK, response)
}

////////////////////////////////////////////////////////////////////////

type createSkillAdminRequest struct {
	SkillName string `json:"skill_name" binding:"required,min=1,max=100"`
}

// Allows admins to manually create new verified skills directly in the system.
func (server *Server) createSkillAdmin(ctx *gin.Context) {
	log.Printf("DEBUG: Starting createSkillAdmin handler")

	var req createSkillAdminRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DEBUG: Create skill admin JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Creating verified skill with name: '%s'", req.SkillName)

	// Normalize skill name (trim whitespace, convert to lowercase for consistency)
	normalizedSkillName := strings.TrimSpace(strings.ToLower(req.SkillName))

	if normalizedSkillName == "" {
		log.Printf("DEBUG: Empty skill name after normalization")
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("skill name cannot be empty")))
		return
	}

	// Check if skill already exists (prevent duplicates)
	existingSkill, err := server.store.GetSkillByName(ctx, normalizedSkillName)
	if err == nil {
		// Skill already exists
		log.Printf("DEBUG: Skill already exists with ID: %d, verified: %v", existingSkill.ID, existingSkill.IsVerified)

		if existingSkill.IsVerified {
			// Already verified - return conflict
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("skill already exists and is verified")))
			return
		} else {
			// Exists but unverified - update to verified instead of creating duplicate
			log.Printf("DEBUG: Updating existing unverified skill to verified")
			updatedSkill, updateErr := server.store.UpdateSkillVerification(ctx, db.UpdateSkillVerificationParams{
				ID:         existingSkill.ID,
				IsVerified: true,
			})
			if updateErr != nil {
				log.Printf("DEBUG: Error updating skill verification: %v", updateErr)
				ctx.JSON(http.StatusInternalServerError, errorResponse(updateErr))
				return
			}

			log.Printf("DEBUG: Successfully updated skill to verified with ID: %d", updatedSkill.ID)
			ctx.JSON(http.StatusOK, updatedSkill) // 200 OK for update
			return
		}
	} else if err != sql.ErrNoRows && err != pgx.ErrNoRows {
		// Database error (not "not found")
		log.Printf("DEBUG: Error checking for existing skill: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// If we reach here, the skill doesn't exist - proceed with creation
	log.Printf("DEBUG: Skill doesn't exist, proceeding with creation")

	// Skill doesn't exist - create new verified skill
	arg := db.CreateSkillParams{
		SkillName:  normalizedSkillName,
		IsVerified: true, // Admin-created skills are verified by default
	}

	skill, err := server.store.CreateSkill(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error creating skill: %v", err)

		// Handle potential duplicate constraint violations at DB level
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("skill name already exists")))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully created verified skill with ID: %d", skill.ID)
	ctx.JSON(http.StatusCreated, skill)

}
////////////////////////////////////////////////////////////////////////
// User Management Handlers
////////////////////////////////////////////////////////////////////////

// Request struct for listing users with pagination and filtering
type listUsersAdminRequest struct {
	PageID   int32  `form:"page_id" binding:"required,min=1"`               // Page number (1-based)
	PageSize int32  `form:"page_size" binding:"required,min=1,max=100"`     // Items per page
	Search   string `form:"search"`                                         // Optional search term
	Role     string `form:"role"`                                           // Optional role filter
}

// GET /admin/users - List and search users with pagination
// Supports searching by name/email and filtering by role
func (server *Server) listUsersAdmin(ctx *gin.Context) {
	var req listUsersAdminRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Validate and prepare role filter
	roleFilterStr := ""
	if req.Role != "" {
		switch req.Role {
		case "admin", "manager", "engineer":
			roleFilterStr = req.Role
		default:
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid role filter")))
			return
		}
	}

	// Prepare search pattern (empty string means no search filter)
	searchPattern := req.Search

	// Use SearchUsers function which handles both search and role filtering efficiently
	// Empty strings are handled by SQL logic: $1::text = '' OR ... AND $2::text = '' OR ...
	users, err := server.store.SearchUsers(ctx, db.SearchUsersParams{
		Column1: searchPattern,  // Search pattern for name/email
		Column2: roleFilterStr,  // Role filter string
		Limit:   req.PageSize,
		Offset:  (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get total count for pagination metadata
	totalCount, err := server.store.CountSearchUsers(ctx, db.CountSearchUsersParams{
		Column1: searchPattern,
		Column2: roleFilterStr,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := gin.H{
		"data":        users,
		"total_count": totalCount,
	}

	ctx.JSON(http.StatusOK, response)
}

// GET /admin/users/:id - Get detailed user information
// Returns user details including team assignment and skills with proficiency levels
func (server *Server) getUserAdmin(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid user ID")))
		return
	}

	// Get user with team information
	user, err := server.store.GetUserWithTeamAndSkills(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get user's skills and proficiency levels
	skills, err := server.store.GetUserSkillsForAdmin(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Construct response with user details and skills
	response := gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"team_id":    user.TeamID,
		"team_name":  user.TeamName,
		"skills":     skills,
	}

	ctx.JSON(http.StatusOK, response)
}

// Request struct for updating user information
type updateUserAdminRequest struct {
	Role   *string `json:"role"`    // Optional role change
	TeamID *int64  `json:"team_id"` // Optional team assignment change
}

// PATCH /admin/users/:id - Update user role or team assignment
// Supports partial updates with comprehensive validation for role changes
func (server *Server) updateUserAdmin(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid user ID")))
		return
	}

	var req updateUserAdminRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get current user information for validation
	currentUser, err := server.store.GetUser(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// CRITICAL: Validate role changes with comprehensive business rules
	if req.Role != nil {
		// Basic role value validation
		switch *req.Role {
		case "admin", "manager", "engineer":
		// Valid roles
		default:
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid role")))
			return
		}

		// Advanced business rule validation using transaction
		validation, err := server.store.ValidateUserRoleChangeTx(ctx, db.ValidateUserRoleChangeTxParams{
			UserID:  id,
			NewRole: db.UserRole(*req.Role),
			TeamID:  req.TeamID,
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		if !validation.IsValid {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New(validation.ErrorMessage)))
			return
		}

		// Handle manager demotion: remove from team management
		if currentUser.Role == db.UserRoleManager && *req.Role != "manager" {
			if validation.ManagedTeam != nil {
				_, err = server.store.SetTeamManager(ctx, db.SetTeamManagerParams{
					ID:        validation.ManagedTeam.ID,
					ManagerID: pgtype.Int8{Valid: false}, // SET NULL
				})
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to remove user from team management")))
					return
				}
			}
		}
	}

	// Prepare update parameters (partial update support)
	updateParams := db.UpdateUserParams{
		ID: id,
	}

	if req.Role != nil {
		updateParams.Role = db.NullUserRole{
			UserRole: db.UserRole(*req.Role),
			Valid:    true,
		}
	}

	if req.TeamID != nil {
		updateParams.TeamID = pgtype.Int8{
			Int64: *req.TeamID,
			Valid: true,
		}
	}

	// Execute the user update
	user, err := server.store.UpdateUser(ctx, updateParams)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Handle manager promotion: assign as team manager
	if req.Role != nil && *req.Role == "manager" && req.TeamID != nil {
		_, err = server.store.SetTeamManager(ctx, db.SetTeamManagerParams{
			ID:        *req.TeamID,
			ManagerID: pgtype.Int8{Int64: id, Valid: true},
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to assign user as team manager")))
			return
		}
	}

	ctx.JSON(http.StatusOK, user)
}

// GET /admin/users/:id/delete-impact - Analyze deletion impact without deleting
// Returns comprehensive impact analysis for admin UI confirmation dialogs
func (server *Server) getUserDeletionImpact(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid user ID")))
		return
	}

	// Execute dry-run deletion impact analysis
	result, err := server.store.GetUserDeletionImpactTx(ctx, db.GetUserDeletionImpactTxParams{
		UserID: id,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("user not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Construct detailed impact response for UI
	response := gin.H{
		"user": gin.H{
			"id":    result.User.ID,
			"name":  result.User.Name,
			"email": result.User.Email,
			"role":  result.User.Role,
		},
		"can_delete": result.CanDelete,
		"impact": gin.H{
			"tasks_to_unassign": gin.H{
				"count": len(result.TasksToUnassign),
				"details": func() []gin.H {
					tasks := make([]gin.H, len(result.TasksToUnassign))
					for i, task := range result.TasksToUnassign {
						tasks[i] = gin.H{
							"id":       task.ID,
							"title":    task.Title,
							"status":   task.Status,
							"priority": task.Priority,
						}
					}
					return tasks
				}(),
			},
			"teams_to_orphan": gin.H{
				"count": len(result.TeamsToOrphan),
				"details": func() []gin.H {
					teams := make([]gin.H, len(result.TeamsToOrphan))
					for i, team := range result.TeamsToOrphan {
						teams[i] = gin.H{
							"id":        team.ID,
							"team_name": team.TeamName,
						}
					}
					return teams
				}(),
			},
			"skills_to_remove":      result.SkillsToRemove,
			"invitations_to_remove": result.InvitationsToRemove,
		},
	}

	// Add blocking reason if deletion is not allowed
	if !result.CanDelete {
		response["blocking_reason"] = result.BlockingReason
	}

	ctx.JSON(http.StatusOK, response)
}

// DELETE /admin/users/:id - Safely delete user with comprehensive cleanup
// Handles all cascading effects according to database schema constraints
func (server *Server) deleteUserAdmin(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid user ID")))
		return
	}

	// Execute safe deletion transaction
	result, err := server.store.SafeDeleteUserTx(ctx, db.SafeDeleteUserTxParams{
		UserID: id,
	})
	if err != nil {
		// Handle business rule violations (e.g., trying to delete admin)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Return comprehensive summary of deletion impact
	response := gin.H{
		"deleted_user":        result.DeletedUser,           // The user that was removed
		"updated_tasks":       len(result.UpdatedTasks),     // Tasks unassigned and reset to "open"
		"updated_teams":       len(result.UpdatedTeams),     // Teams that became unmanaged
		"removed_skills":      result.RemovedSkills,         // User-skill associations removed
		"removed_invitations": result.RemovedInvitations,    // Invitations sent by user removed
	}

	ctx.JSON(http.StatusOK, response)
}
