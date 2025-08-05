// api/manager_handler.go
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
)

////////////////////////////////////////////////////////////////////////
// Dashboard and Team Management
////////////////////////////////////////////////////////////////////////

// getDashboardStats provides a single endpoint for all dashboard statistics
func (server *Server) getDashboardStats(ctx *gin.Context) {
	log.Printf("DEBUG: Starting getDashboardStats handler")

	// Get authorization payload
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for dashboard stats: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for dashboard stats")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Getting dashboard stats for team ID: %d", teamID)

	// Get active projects count
	activeProjects, err := server.store.CountActiveProjectsByTeam(ctx, teamID)
	if err != nil {
		log.Printf("DEBUG: Error counting active projects: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get open tasks count
	openTasks, err := server.store.CountOpenTasksByTeam(ctx, teamID)
	if err != nil {
		log.Printf("DEBUG: Error counting open tasks: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get available engineers count
	availableEngineers, err := server.store.CountUsersByTeamAndAvailability(ctx, db.CountUsersByTeamAndAvailabilityParams{
		TeamID:       pgtype.Int8{Int64: teamID, Valid: true},
		Availability: db.AvailabilityStatusAvailable,
	})
	if err != nil {
		log.Printf("DEBUG: Error counting available engineers: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get total engineers count
	totalEngineers, err := server.store.CountUsersByTeamAndRole(ctx, db.CountUsersByTeamAndRoleParams{
		TeamID: pgtype.Int8{Int64: teamID, Valid: true},
		Role:   db.UserRoleEngineer,
	})
	if err != nil {
		log.Printf("DEBUG: Error counting total engineers: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := gin.H{
		"active_projects":     activeProjects,
		"open_tasks":          openTasks,
		"available_engineers": availableEngineers,
		"total_engineers":     totalEngineers,
	}

	log.Printf("DEBUG: Dashboard stats - Projects: %d, Tasks: %d, Available: %d, Total: %d", 
		activeProjects, openTasks, availableEngineers, totalEngineers)

	ctx.JSON(http.StatusOK, response)
}

// getTeamMembers lists all engineers on the manager's team with availability status
func (server *Server) getTeamMembers(ctx *gin.Context) {
	log.Printf("DEBUG: Starting getTeamMembers handler")

	// Get authorization payload
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for team members: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for team members")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Getting team members for team ID: %d", teamID)

	// Get all engineers in the team
	engineers, err := server.store.ListEngineersByTeam(ctx, pgtype.Int8{Int64: teamID, Valid: true})
	if err != nil {
		log.Printf("DEBUG: Error listing engineers by team: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Convert to response format
	type teamMemberResponse struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		Email        string `json:"email"`
		Availability string `json:"availability"`
	}

	members := make([]teamMemberResponse, 0, len(engineers))
	for _, engineer := range engineers {
		members = append(members, teamMemberResponse{
			ID:           engineer.ID,
			Name:         engineer.Name.String,
			Email:        engineer.Email,
			Availability: string(engineer.Availability),
		})
	}

	log.Printf("DEBUG: Found %d engineers in team %d", len(members), teamID)
	ctx.JSON(http.StatusOK, members)
}

////////////////////////////////////////////////////////////////////////
// Invitation Handler (for Managers)
////////////////////////////////////////////////////////////////////////

type inviteEngineerRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// inviteEngineer handles creating invitations for engineer role by managers
func (server *Server) inviteEngineer(ctx *gin.Context) {
	log.Printf("DEBUG: Starting inviteEngineer handler")

	var req inviteEngineerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DEBUG: Invite engineer JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Creating engineer invitation - Email: %s", req.Email)

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for engineer invitation: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract user_id with type assertion
	userIDFloat, ok := authPayload["user_id"].(float64)
	if !ok {
		log.Printf("DEBUG: user_id not found or not a float64 in auth payload for engineer invitation. Payload: %+v", authPayload)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("invalid user_id in token")))
		return
	}

	inviterID := int64(userIDFloat)
	log.Printf("DEBUG: Extracted Manager ID: %d", inviterID)

	// For engineer invitations by managers, team_id is auto-derived from manager's team
	// No need to specify TeamID in params - the transaction will handle it
	arg := db.CreateInvitationTxParams{
		InviterID:     inviterID,
		EmailToInvite: req.Email,
		RoleToInvite:  db.UserRoleEngineer,
		// TeamID is intentionally omitted - will be auto-derived from manager's team
	}

	log.Printf("DEBUG: Calling CreateInvitationTx with params: %+v", arg)

	result, err := server.store.CreateInvitationTx(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error creating engineer invitation: %v", err)

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
		case errors.Is(err, db.ErrManagerMustHaveTeam):
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		default:
			// Generic database or system error
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	log.Printf("DEBUG: Successfully created engineer invitation with ID: %d, Token: %s, Expires: %v", 
		result.Invitation.ID, result.Invitation.InvitationToken, result.Invitation.ExpiresAt.Time)

	// Return the created invitation details
	ctx.JSON(http.StatusCreated, result.Invitation)
}

type listSentInvitationsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=20"`
}

// listSentInvitations handles retrieving invitations sent by the current manager
func (server *Server) listSentInvitations(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listSentInvitations handler")

	var req listSentInvitationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		log.Printf("DEBUG: List sent invitations query bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: List sent invitations request params - PageID: %d, PageSize: %d", req.PageID, req.PageSize)

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for listing invitations: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract user_id with type assertion
	userIDFloat, ok := authPayload["user_id"].(float64)
	if !ok {
		log.Printf("DEBUG: user_id not found or not a float64 in auth payload for listing invitations. Payload: %+v", authPayload)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("invalid user_id in token")))
		return
	}

	inviterID := int64(userIDFloat)
	log.Printf("DEBUG: Extracted Manager ID: %d", inviterID)

	// Query invitations sent by this manager
	invitations, err := server.store.ListInvitationsByInviter(ctx, db.ListInvitationsByInviterParams{
		InviterID: inviterID,
		Limit:     req.PageSize,
		Offset:    (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		log.Printf("DEBUG: Error listing invitations by inviter: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get total count for pagination metadata
	totalCount, err := server.store.CountInvitationsByInviter(ctx, inviterID)
	if err != nil {
		log.Printf("DEBUG: Error counting invitations by inviter: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Retrieved %d invitations sent by manager, total count: %d", len(invitations), totalCount)

	// Convert to the unified response struct for API consistency
	finalInvitations := make([]invitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		finalInvitations = append(finalInvitations, invitationResponse{
			ID:           inv.ID,
			Email:        inv.Email,
			RoleToInvite: inv.RoleToInvite,
			Status:       inv.Status,
			InviterName:  inv.InviterName,
			InviterRole:  inv.InviterRole, // This is now string type consistently
			CreatedAt:    inv.CreatedAt,
		})
	}

	rsp := paginatedResponse[invitationResponse]{
		TotalCount: totalCount,
		Data:       finalInvitations,
	}

	log.Printf("DEBUG: Successfully returning %d invitations with pagination", len(finalInvitations))
	ctx.JSON(http.StatusOK, rsp)
}

type cancelInvitationRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// cancelInvitation handles canceling pending invitations sent by the current manager
func (server *Server) cancelInvitation(ctx *gin.Context) {
	log.Printf("DEBUG: Starting cancelInvitation handler")

	var req cancelInvitationRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		log.Printf("DEBUG: Cancel invitation URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Canceling invitation with ID: %d", req.ID)

	// Get authorization payload
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for canceling invitation: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	userIDFloat, ok := authPayload["user_id"].(float64)
	if !ok {
		log.Printf("DEBUG: user_id not found in auth payload for canceling invitation")
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("invalid user_id in token")))
		return
	}

	managerID := int64(userIDFloat)
	log.Printf("DEBUG: Extracted Manager ID: %d", managerID)

	// First, check if the invitation exists and verify ownership
	invitation, err := server.store.GetInvitationByID(ctx, req.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("DEBUG: Invitation not found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("invitation not found")))
			return
		}
		log.Printf("DEBUG: Error checking invitation: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify that this manager sent the invitation
	if invitation.InviterID != managerID {
		log.Printf("DEBUG: Manager %d attempted to cancel invitation %d sent by %d", managerID, req.ID, invitation.InviterID)
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("you can only cancel invitations you sent")))
		return
	}

	// Check if invitation can be canceled
	if invitation.Status != "pending" {
		log.Printf("DEBUG: Cannot cancel invitation with status: %s", invitation.Status)
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("only pending invitations can be canceled")))
		return
	}

	// Proceed with deletion
	err = server.store.DeleteInvitation(ctx, req.ID)
	if err != nil {
		log.Printf("DEBUG: Error deleting invitation: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully canceled invitation with ID: %d", req.ID)
	ctx.Status(http.StatusNoContent)
}

////////////////////////////////////////////////////////////////////////
// Project Handler (for Managers) - Enhanced with Task Counts
////////////////////////////////////////////////////////////////////////

type createProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

// createProject handles creating a new project by manager users
func (server *Server) createProject(ctx *gin.Context) {
	log.Printf("DEBUG: Starting createProject handler")

	var req createProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DEBUG: Create project JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Creating project - Name: '%s', Description: '%s'", req.Name, req.Description)

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for project creation: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract team_id with type assertion
	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team. Auth payload: %+v", authPayload)
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Extracted Team ID: %d", teamID)

	arg := db.CreateProjectParams{
		ProjectName: req.Name,
		TeamID:      teamID,
		Description: pgtype.Text{String: req.Description, Valid: true},
	}

	log.Printf("DEBUG: Calling CreateProject with params: %+v", arg)

	project, err := server.store.CreateProject(ctx, arg)
	if err != nil {
		log.Printf("DEBUG: Error creating project: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully created project with ID: %d", project.ID)
	ctx.JSON(http.StatusCreated, project)
}

type listProjectsRequest struct {
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=50"`
	Archived *bool  `form:"archived"`  // Optional: true = archived only, false/nil = active only
}

// Enhanced project response with task counts
type projectWithTaskCounts struct {
	db.Project
	TotalTasks     int64 `json:"total_tasks"`
	CompletedTasks int64 `json:"completed_tasks"`
}

// listProjects handles retrieving projects with archive filtering and task counts
func (server *Server) listProjects(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listProjects handler")

	var req listProjectsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		log.Printf("DEBUG: List projects query bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: List projects request params - PageID: %d, PageSize: %d, Archived: %v", 
		req.PageID, req.PageSize, req.Archived)

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for listing projects: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract team_id with type assertion
	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for listing projects")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Extracted Team ID: %d", teamID)

	var projects []db.Project
	var totalCount int64

	// Default to showing active projects unless specifically requesting archived ones
	if req.Archived != nil && *req.Archived {
		// Show archived projects
		archivedParams := db.ListArchivedProjectsByTeamParams{
			TeamID: teamID,
			Limit:  req.PageSize,
			Offset: (req.PageID - 1) * req.PageSize,
		}
		projects, err = server.store.ListArchivedProjectsByTeam(ctx, archivedParams)
		if err == nil {
			totalCount, err = server.store.CountArchivedProjectsByTeam(ctx, teamID)
		}
		log.Printf("DEBUG: Listing archived projects")
	} else {
		// Show active projects (default)
		activeParams := db.ListActiveProjectsByTeamParams{
			TeamID: teamID,
			Limit:  req.PageSize,
			Offset: (req.PageID - 1) * req.PageSize,
		}
		projects, err = server.store.ListActiveProjectsByTeam(ctx, activeParams)
		if err == nil {
			totalCount, err = server.store.CountActiveProjectsByTeam(ctx, teamID)
		}
		log.Printf("DEBUG: Listing active projects")
	}

	if err != nil {
		log.Printf("DEBUG: Error listing projects: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Enhance projects with task counts
	enhancedProjects := make([]projectWithTaskCounts, 0, len(projects))
	for _, project := range projects {
		projectID := pgtype.Int8{Int64: project.ID, Valid: true}
		
		// Get total active tasks count
		totalTasks, err := server.store.CountActiveTasksByProject(ctx, projectID)
		if err != nil {
			log.Printf("DEBUG: Error counting tasks for project %d: %v", project.ID, err)
			totalTasks = 0 // Continue with 0 if error
		}

		// Get completed tasks count
		completedTasks, err := server.store.CountTasksByProjectAndStatus(ctx, db.CountTasksByProjectAndStatusParams{
			ProjectID: projectID,
			Status:    db.TaskStatusDone,
		})
		if err != nil {
			log.Printf("DEBUG: Error counting completed tasks for project %d: %v", project.ID, err)
			completedTasks = 0 // Continue with 0 if error
		}

		enhancedProjects = append(enhancedProjects, projectWithTaskCounts{
			Project:        project,
			TotalTasks:     totalTasks,
			CompletedTasks: completedTasks,
		})
	}

	log.Printf("DEBUG: Retrieved %d projects for team %d, total count: %d", len(enhancedProjects), teamID, totalCount)

	rsp := paginatedResponse[projectWithTaskCounts]{
		TotalCount: totalCount,
		Data:       enhancedProjects,
	}

	ctx.JSON(http.StatusOK, rsp)
}

type getProjectRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// getProject handles retrieving a specific project by ID (team-scoped)
func (server *Server) getProject(ctx *gin.Context) {
	log.Printf("DEBUG: Starting getProject handler")

	var req getProjectRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		log.Printf("DEBUG: Get project URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Getting project with ID: %d", req.ID)

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for getting project: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract team_id with type assertion
	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for getting project")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Extracted Team ID: %d", teamID)

	// Use team-scoped project retrieval to ensure manager can only access their team's projects
	project, err := server.store.GetProjectByIDAndTeam(ctx, db.GetProjectByIDAndTeamParams{
		ID:     req.ID,
		TeamID: teamID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("DEBUG: Project not found or doesn't belong to manager's team")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("project not found")))
			return
		}
		log.Printf("DEBUG: Error getting project by ID and team: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully retrieved project: %s", project.ProjectName)
	ctx.JSON(http.StatusOK, project)
}

type updateProjectRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateProjectBody struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// updateProject handles updating a project's name and/or description
func (server *Server) updateProject(ctx *gin.Context) {
	log.Printf("DEBUG: Starting updateProject handler")

	var uriReq updateProjectRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		log.Printf("DEBUG: Update project URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var bodyReq updateProjectBody
	if err := ctx.ShouldBindJSON(&bodyReq); err != nil {
		log.Printf("DEBUG: Update project JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Updating project ID: %d", uriReq.ID)

	// Validate that at least one field is being updated
	if bodyReq.Name == nil && bodyReq.Description == nil {
		log.Printf("DEBUG: No fields provided for update")
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("at least one field (name or description) must be provided")))
		return
	}

	// Get authorization payload with proper error handling
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for updating project: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	// Safely extract team_id with type assertion
	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for updating project")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Extracted Team ID: %d", teamID)

	// First, verify the project exists and belongs to the manager's team
	existingProject, err := server.store.GetProjectByIDAndTeam(ctx, db.GetProjectByIDAndTeamParams{
		ID:     uriReq.ID,
		TeamID: teamID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("DEBUG: Project not found or doesn't belong to manager's team for update")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("project not found")))
			return
		}
		log.Printf("DEBUG: Error checking project ownership for update: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Check if project is archived - cannot update archived projects
	if existingProject.Archived {
		log.Printf("DEBUG: Attempted to update archived project")
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("cannot update archived projects")))
		return
	}

	// Prepare update parameters
	updateParams := db.UpdateProjectParams{
		ID:     uriReq.ID,
		TeamID: teamID,
	}

	// Set project name (use new value if provided, otherwise use existing)
	if bodyReq.Name != nil {
		updateParams.ProjectName = *bodyReq.Name
		log.Printf("DEBUG: Updating project name to: %s", *bodyReq.Name)
	} else {
		updateParams.ProjectName = existingProject.ProjectName
		log.Printf("DEBUG: Keeping existing project name: %s", existingProject.ProjectName)
	}

	// Set description (use new value if provided, otherwise use existing)
	if bodyReq.Description != nil {
		updateParams.Description = pgtype.Text{String: *bodyReq.Description, Valid: true}
		log.Printf("DEBUG: Updating project description")
	} else {
		updateParams.Description = existingProject.Description
		log.Printf("DEBUG: Keeping existing project description")
	}

	// Execute the update
	updatedProject, err := server.store.UpdateProject(ctx, updateParams)
	if err != nil {
		log.Printf("DEBUG: Error updating project: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Successfully updated project with ID: %d", updatedProject.ID)
	ctx.JSON(http.StatusOK, updatedProject)
}

type archiveProjectRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// archiveProject handles archiving a project and all its tasks
func (server *Server) archiveProject(ctx *gin.Context) {
	log.Printf("DEBUG: Starting archiveProject handler")

	var req archiveProjectRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		log.Printf("DEBUG: Archive project URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Archiving project with ID: %d", req.ID)

	// Get authorization payload
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for archiving project: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for archiving project")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)
	log.Printf("DEBUG: Extracted Team ID: %d", teamID)

	// Archive the project and all its tasks using the transaction
	result, err := server.store.ArchiveProjectTx(ctx, db.ArchiveProjectTxParams{
		ProjectID: req.ID,
		TeamID:    teamID,
	})
	if err != nil {
		log.Printf("DEBUG: Error archiving project: %v", err)
		
		switch {
		case errors.Is(err, db.ErrProjectNotFound):
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		case errors.Is(err, db.ErrProjectAlreadyArchived):
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		default:
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}
	}

	log.Printf("DEBUG: Successfully archived project with ID: %d and %d tasks", 
		result.ArchivedProject.ID, result.ArchivedTasksCount)

	// Return result with both project and task count
	response := gin.H{
		"archived_project":     result.ArchivedProject,
		"archived_tasks_count": result.ArchivedTasksCount,
		"message":             fmt.Sprintf("Project and %d tasks archived successfully", result.ArchivedTasksCount),
	}
	
	ctx.JSON(http.StatusOK, response)
}

////////////////////////////////////////////////////////////////////////
// Task Handler (for Managers) - Enhanced with Project Tasks and Update
////////////////////////////////////////////////////////////////////////

type createTaskRequest struct {
	ProjectID   int64  `json:"project_id" binding:"required,min=1"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Priority    string `json:"priority" binding:"required,oneof=low medium high critical"`
}

func (server *Server) createTask(ctx *gin.Context) {
	var req createTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload, _ := getAuthorizationPayload(ctx)
	managerTeamID, ok := authPayload["team_id"].(float64)
	if !ok || managerTeamID == 0 {
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Validate project belongs to manager's team and is not archived
	project, err := server.store.GetProjectByIDAndTeam(ctx, db.GetProjectByIDAndTeamParams{
		ID:     req.ProjectID,
		TeamID: int64(managerTeamID),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("project not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Cannot create tasks in archived projects
	if project.Archived {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("cannot create tasks in archived projects")))
		return
	}

	requiredSkills, err := server.skillzProcessor.ExtractAndNormalize(ctx, req.Description)
	if err != nil {
		log.Printf("❌ skillzProcessor error during task creation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not process task description for skills"})
		return
	}

	arg := db.ProcessNewTaskTxParams{
		CreateTaskParams: db.CreateTaskParams{
			ProjectID:   pgtype.Int8{Int64: req.ProjectID, Valid: true},
			Title:       req.Title,
			Description: pgtype.Text{String: req.Description, Valid: true},
			Status:      db.TaskStatusOpen,
			Priority:    db.TaskPriority(req.Priority),
		},
		RequiredSkillNames: requiredSkills,
	}

	result, err := server.store.ProcessNewTask(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, result)
}

type listProjectTasksURIRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type listProjectTasksQueryRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=100"`
}

// listProjectTasks gets all tasks for a specific project with assignee names
func (server *Server) listProjectTasks(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listProjectTasks handler")

	// Bind URI parameters
	var uriReq listProjectTasksURIRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		log.Printf("DEBUG: List project tasks URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	
	// Bind query parameters
	var queryReq listProjectTasksQueryRequest
	if err := ctx.ShouldBindQuery(&queryReq); err != nil {
		log.Printf("DEBUG: List project tasks query bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Getting tasks for project ID: %d, PageID: %d, PageSize: %d", 
		uriReq.ID, queryReq.PageID, queryReq.PageSize)

	// Get authorization payload
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for project tasks: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for project tasks")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)

	// Validate project belongs to manager's team
	_, err = server.store.GetProjectByIDAndTeam(ctx, db.GetProjectByIDAndTeamParams{
		ID:     uriReq.ID, // Use uriReq.ID instead of req.ID
		TeamID: teamID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("DEBUG: Project not found or doesn't belong to manager's team")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("project not found")))
			return
		}
		log.Printf("DEBUG: Error validating project ownership: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get tasks with assignee names
	tasks, err := server.store.ListTasksWithAssigneeNames(ctx, db.ListTasksWithAssigneeNamesParams{
		ProjectID: pgtype.Int8{Int64: uriReq.ID, Valid: true}, // Use uriReq.ID
		Limit:     queryReq.PageSize,                          // Use queryReq.PageSize
		Offset:    (queryReq.PageID - 1) * queryReq.PageSize,  // Use queryReq values
	})
	if err != nil {
		log.Printf("DEBUG: Error listing tasks with assignee names: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Convert to response format (rest of the function remains the same)
	type taskWithAssigneeResponse struct {
		ID           int64                  `json:"id"`
		Title        string                 `json:"title"`
		Status       db.TaskStatus          `json:"status"`
		Priority     db.TaskPriority        `json:"priority"`
		AssigneeID   *int64                 `json:"assignee_id"`
		AssigneeName *string                `json:"assignee_name"`
	}

	taskResponses := make([]taskWithAssigneeResponse, 0, len(tasks))
	for _, task := range tasks {
		response := taskWithAssigneeResponse{
			ID:       task.ID,
			Title:    task.Title,
			Status:   task.Status,
			Priority: task.Priority,
		}

		if task.AssigneeID.Valid {
			response.AssigneeID = &task.AssigneeID.Int64
		}

		if task.AssigneeName.Valid {
			response.AssigneeName = &task.AssigneeName.String
		}

		taskResponses = append(taskResponses, response)
	}

	log.Printf("DEBUG: Retrieved %d tasks for project %d", len(taskResponses), uriReq.ID)
	ctx.JSON(http.StatusOK, taskResponses)
}

type updateTaskRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type updateTaskBody struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Priority    *string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	Status      *string `json:"status" binding:"omitempty,oneof=open in_progress done"`
	AssigneeID  *int64  `json:"assignee_id"`
}

// updateTask handles updating task details, status, or assignment
func (server *Server) updateTask(ctx *gin.Context) {
	log.Printf("DEBUG: Starting updateTask handler")

	var uriReq updateTaskRequest
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		log.Printf("DEBUG: Update task URI bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var bodyReq updateTaskBody
	if err := ctx.ShouldBindJSON(&bodyReq); err != nil {
		log.Printf("DEBUG: Update task JSON bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Updating task ID: %d", uriReq.ID)

	// Validate that at least one field is being updated
	if bodyReq.Title == nil && bodyReq.Description == nil && bodyReq.Priority == nil && 
	   bodyReq.Status == nil && bodyReq.AssigneeID == nil {
		log.Printf("DEBUG: No fields provided for task update")
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("at least one field must be provided for update")))
		return
	}

	// Get authorization payload
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		log.Printf("DEBUG: Failed to get authorization payload for updating task: %v", err)
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("unauthorized")))
		return
	}

	teamIDFloat, ok := authPayload["team_id"].(float64)
	if !ok || teamIDFloat == 0 {
		log.Printf("DEBUG: Manager is not assigned to a team for updating task")
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	teamID := int64(teamIDFloat)

	// Get existing task and validate ownership
	existingTask, err := server.store.GetTask(ctx, uriReq.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("DEBUG: Task not found")
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("task not found")))
			return
		}
		log.Printf("DEBUG: Error getting existing task: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Validate task belongs to manager's team via project
	project, err := server.store.GetProject(ctx, existingTask.ProjectID.Int64)
	if err != nil {
		log.Printf("DEBUG: Error getting task's project: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if project.TeamID != teamID {
		log.Printf("DEBUG: Task does not belong to manager's team")
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("task does not belong to your team")))
		return
	}

	// Cannot update archived tasks
	if existingTask.Archived {
		log.Printf("DEBUG: Attempted to update archived task")
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("cannot update archived tasks")))
		return
	}

	// If assigning to someone, validate assignee belongs to team
	if bodyReq.AssigneeID != nil && *bodyReq.AssigneeID != 0 {
		assignee, err := server.store.GetUser(ctx, *bodyReq.AssigneeID)
		if err != nil {
			log.Printf("DEBUG: Error getting assignee: %v", err)
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid assignee")))
			return
		}

		if !assignee.TeamID.Valid || assignee.TeamID.Int64 != teamID {
			log.Printf("DEBUG: Assignee does not belong to manager's team")
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("assignee must be from your team")))
			return
		}

		if assignee.Role != db.UserRoleEngineer {
			log.Printf("DEBUG: Assignee is not an engineer")
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("can only assign tasks to engineers")))
			return
		}
	}

	// Prepare update parameters
	updateParams := db.UpdateTaskParams{
		ID: uriReq.ID,
	}

	// Update fields based on what was provided
	if bodyReq.Title != nil {
		updateParams.Title = pgtype.Text{String: *bodyReq.Title, Valid: true}
		log.Printf("DEBUG: Updating task title")
	}

	if bodyReq.Description != nil {
		updateParams.Description = pgtype.Text{String: *bodyReq.Description, Valid: true}
		log.Printf("DEBUG: Updating task description")
	}

	if bodyReq.Priority != nil {
		updateParams.Priority = db.NullTaskPriority{TaskPriority: db.TaskPriority(*bodyReq.Priority), Valid: true}
		log.Printf("DEBUG: Updating task priority to: %s", *bodyReq.Priority)
	}

	if bodyReq.Status != nil {
		updateParams.Status = db.NullTaskStatus{TaskStatus: db.TaskStatus(*bodyReq.Status), Valid: true}
		log.Printf("DEBUG: Updating task status to: %s", *bodyReq.Status)

		// Set completion time if marking as done
		if *bodyReq.Status == "done" {
			updateParams.CompletedAt = pgtype.Timestamp{Time: time.Now(), Valid: true}
		}
	}

	if bodyReq.AssigneeID != nil {
		if *bodyReq.AssigneeID == 0 {
			// Unassign task
			updateParams.AssigneeID = pgtype.Int8{Valid: false}
			log.Printf("DEBUG: Unassigning task")
		} else {
			// Assign task
			updateParams.AssigneeID = pgtype.Int8{Int64: *bodyReq.AssigneeID, Valid: true}
			log.Printf("DEBUG: Assigning task to user %d", *bodyReq.AssigneeID)
		}
	}

	// Handle user availability updates based on assignment changes
	var oldAssigneeID, newAssigneeID *int64

	if existingTask.AssigneeID.Valid {
		oldAssigneeID = &existingTask.AssigneeID.Int64
	}

	if bodyReq.AssigneeID != nil {
		if *bodyReq.AssigneeID != 0 {
			newAssigneeID = bodyReq.AssigneeID
		}
	} else if existingTask.AssigneeID.Valid {
		// Keep existing assignee if not changing
		newAssigneeID = &existingTask.AssigneeID.Int64
	}

	// Execute the task update
	updatedTask, err := server.store.UpdateTask(ctx, updateParams)
	if err != nil {
		log.Printf("DEBUG: Error updating task: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Update user availability if assignment changed
	if oldAssigneeID != nil && (newAssigneeID == nil || *oldAssigneeID != *newAssigneeID) {
		// Old assignee should become available
		_, err = server.store.UpdateUser(ctx, db.UpdateUserParams{
			ID:           *oldAssigneeID,
			Availability: db.NullAvailabilityStatus{AvailabilityStatus: db.AvailabilityStatusAvailable, Valid: true},
		})
		if err != nil {
			log.Printf("DEBUG: Error updating old assignee availability: %v", err)
			// Continue - don't fail the whole operation
		}
	}

	if newAssigneeID != nil && (oldAssigneeID == nil || *oldAssigneeID != *newAssigneeID) {
		// New assignee should become busy
		_, err = server.store.UpdateUser(ctx, db.UpdateUserParams{
			ID:           *newAssigneeID,
			Availability: db.NullAvailabilityStatus{AvailabilityStatus: db.AvailabilityStatusBusy, Valid: true},
		})
		if err != nil {
			log.Printf("DEBUG: Error updating new assignee availability: %v", err)
			// Continue - don't fail the whole operation
		}
	}

	log.Printf("DEBUG: Successfully updated task with ID: %d", updatedTask.ID)
	ctx.JSON(http.StatusOK, updatedTask)
}

////////////////////////////////////////////////////////////////////////
// Recommendation Handler (for Managers)
////////////////////////////////////////////////////////////////////////

type getRecommendationsRequest struct {
	TaskID int64 `json:"task_id" binding:"required,min=1"`
	Limit  int   `json:"limit,omitempty"`
}

type recommenderAPIRequest struct {
	SkillIDs []int32 `json:"skill_ids"`
	Limit    int     `json:"limit"`
}

type recommenderAPIResponse struct {
	Recommendations []struct {
		UserID int64   `json:"user_id"`
		Score  float64 `json:"score"`
	} `json:"recommendations"`
}

type EnrichedRecommendation struct {
	UserID int64   `json:"user_id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Score  float64 `json:"score"`
}

func (server *Server) getRecommendations(ctx *gin.Context) {
	var req getRecommendationsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("ERROR: Bind error: %v", err)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Getting recommendations for task ID: %d", req.TaskID)
	log.Printf("DEBUG: Recommender API URL: %s", server.config.RecommenderAPIURL)
	log.Printf("DEBUG: Recommender API Key exists: %t", server.config.RecommenderAPIKey != "")

	authPayload, _ := getAuthorizationPayload(ctx)
	managerTeamID, ok := authPayload["team_id"].(float64)
	if !ok || managerTeamID == 0 {
		err := errors.New("forbidden: manager is not assigned to a team")
		log.Printf("ERROR: %v", err)
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Manager team ID: %v", managerTeamID)

	task, err := server.store.GetTask(ctx, req.TaskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Printf("ERROR: Task not found: %d", req.TaskID)
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("task not found")))
			return
		}
		log.Printf("ERROR: GetTask failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Found task: %+v", task)

	project, err := server.store.GetProject(ctx, task.ProjectID.Int64)
	if err != nil {
		log.Printf("ERROR: GetProject failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Found project: %+v", project)

	if project.TeamID != int64(managerTeamID) {
		err := errors.New("forbidden: this task does not belong to your team")
		log.Printf("ERROR: %v (project team: %d, manager team: %v)", err, project.TeamID, managerTeamID)
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	requiredSkills, err := server.store.GetSkillsForTask(ctx, req.TaskID)
	if err != nil {
		log.Printf("ERROR: GetSkillsForTask failed: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	log.Printf("DEBUG: Found %d skills for task", len(requiredSkills))

	if len(requiredSkills) == 0 {
		log.Printf("DEBUG: No skills found, returning empty recommendations")
		ctx.JSON(http.StatusOK, gin.H{"recommendations": []EnrichedRecommendation{}})
		return
	}

	var skillIDs []int32
	for _, skill := range requiredSkills {
		skillIDs = append(skillIDs, int32(skill.ID))
	}

	log.Printf("DEBUG: Skill IDs: %v", skillIDs)

	limit := 10
	if req.Limit > 0 && req.Limit <= 50 {
		limit = req.Limit
	}

	recommenderReqPayload := recommenderAPIRequest{SkillIDs: skillIDs, Limit: limit}
	recommenderBody, _ := json.Marshal(recommenderReqPayload)

	log.Printf("DEBUG: Calling recommender API with payload: %s", string(recommenderBody))

	request, err := http.NewRequest("POST", server.config.RecommenderAPIURL, bytes.NewBuffer(recommenderBody))
	if err != nil {
		log.Printf("ERROR: Failed to create request: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Internal-API-Key", server.config.RecommenderAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("ERROR: HTTP request failed: %v", err)
		ctx.JSON(http.StatusServiceUnavailable, errorResponse(errors.New("recommendation service is unavailable")))
		return
	}
	defer response.Body.Close()

	bodyBytes, _ := io.ReadAll(response.Body)
	log.Printf("DEBUG: Recommender API response status: %d", response.StatusCode)
	log.Printf("DEBUG: Recommender API response body: %s", string(bodyBytes))

	if response.StatusCode != http.StatusOK {
		errText := fmt.Sprintf("recommendation service failed: %s", string(bodyBytes))
		log.Printf("ERROR: %s", errText)
		ctx.JSON(http.StatusServiceUnavailable, errorResponse(errors.New(errText)))
		return
	}

	// Reset body reader for JSON decoding
	var recommenderResp recommenderAPIResponse
	if err := json.Unmarshal(bodyBytes, &recommenderResp); err != nil {
		log.Printf("ERROR: Failed to parse JSON response: %v", err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to parse recommendation response")))
		return
	}

	log.Printf("DEBUG: Parsed %d recommendations from API", len(recommenderResp.Recommendations))

	var enrichedRecommendations []EnrichedRecommendation
	for _, rec := range recommenderResp.Recommendations {
		user, err := server.store.GetUser(ctx, rec.UserID)
		if err == nil && user.TeamID.Int64 == int64(managerTeamID) {
			enrichedRecommendations = append(enrichedRecommendations, EnrichedRecommendation{
				UserID: user.ID,
				Name:   user.Name.String,
				Email:  user.Email,
				Score:  rec.Score,
			})
			log.Printf("DEBUG: Added recommendation for user %d (%s)", user.ID, user.Name.String)
		} else if err != nil {
			log.Printf("DEBUG: Failed to get user %d: %v", rec.UserID, err)
		} else {
			log.Printf("DEBUG: User %d not in same team (user team: %d, manager team: %v)", rec.UserID, user.TeamID.Int64, managerTeamID)
		}
	}

	log.Printf("DEBUG: Returning %d enriched recommendations", len(enrichedRecommendations))
	ctx.JSON(http.StatusOK, gin.H{"recommendations": enrichedRecommendations})
}
