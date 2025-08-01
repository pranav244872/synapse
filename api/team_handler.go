// api/team_handler.go
package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
)

// createTeamRequest defines the JSON payload for creating a team.
type createTeamRequest struct {
	TeamName string `json:"team_name" binding:"required"`
}

// createTeam handles the creation of a new team.
// It is a protected endpoint accessible only by users with the "admin" role.
func (server *Server) createTeam(ctx *gin.Context) {
	var req createTeamRequest

	// Step 1: Bind and validate the incoming JSON request.
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Step 2: Get the authorization payload from the JWT middleware.
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// Step 3: Authorize the user. Only admins can create teams.
	role, _ := authPayload["role"].(string)
	if role != string(db.UserRoleAdmin) {
		err := errors.New("forbidden: only admins can create teams")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Step 4: Call the database to create the team.
	// The manager_id is intentionally left null; it gets assigned when a manager accepts an invitation.
	arg := db.CreateTeamParams{
		TeamName:  req.TeamName,
		ManagerID: pgtype.Int8{Valid: false},
	}

	team, err := server.store.CreateTeam(ctx, arg)
	if err != nil {
		// Handle potential database errors, such as a unique constraint violation on team_name.
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Step 5: Return the newly created team object with a 201 Created status.
	ctx.JSON(http.StatusCreated, team)
}
