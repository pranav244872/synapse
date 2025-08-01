// api/project_handler.go

package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
)

////////////////////////////////////////////////////////////////////////
// Protected Endpoint: Create a New Project
////////////////////////////////////////////////////////////////////////

// createProjectRequest defines the JSON payload for creating a project.
// Example:
// {
//   	"name": "Project Synapse MVP"
//		"description" : "For clients in the USA"
// }
type createProjectRequest struct {
	Name string `json:"name" binding:"required"` // The name for the new project.
	Description string `json:"description" binding:"required"` // The description of the new project
}

// It is a protected endpoint accessible only by users with the "manager" role.
// createProject handles the creation of a new project, scoped to the manager's team.
func (server *Server) createProject(ctx *gin.Context) {
	var req createProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get payload from JWT (contains user_id, role, team_id).
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Authorize: Only managers can create projects.
	role, _ := authPayload["role"].(string)
	if role != string(db.UserRoleManager) {
		err := errors.New("forbidden: only managers can create new projects")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Get the manager's team ID from the token payload.
	teamID, ok := authPayload["team_id"].(float64)
	if !ok || teamID == 0 {
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Prepare parameters for the database call.
	// This now correctly uses the CreateProjectParams struct.
	arg := db.CreateProjectParams{
		ProjectName: req.Name,
		Description: pgtype.Text{String: req.Description, Valid: true},
		TeamID:      int64(teamID),
	}

	// Call the database to create the project.
	project, err := server.store.CreateProject(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, project)
}
