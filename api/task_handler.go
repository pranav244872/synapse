// api/task_handler.api
package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/pranav244872/synapse/db/sqlc"
)

////////////////////////////////////////////////////////////////////////
// Protected Endpoint: Create a New Task
////////////////////////////////////////////////////////////////////////

// createTaskRequest defines the JSON payload for creating a new task.
type createTaskRequest struct {
	ProjectID   int64  `json:"project_id" binding:"required,min=1"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Priority    string `json:"priority,omitempty" binding:"required,oneof=low medium high critical"`
}

// createTask handles the creation of a new task with team based validation.
// It is a protected endpoint accessible only by users with the "manager" role.
func (server *Server) createTask(ctx *gin.Context) {
	var req createTaskRequest

	// Step 1: Bind and validate the request body
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Step 2: Authorize the user. Only managers can create tasks.
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		// This should technically not happen if authMiddleware is working
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}
	
	// Authorize: Only Managers can create tasks
	role, ok := authPayload["role"].(string)
	if !ok || role != string(db.UserRoleManager) {
		err := errors.New("forbidden: only managers can create new tasks")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Get the manager's team ID from the token payload.
	managerTeamID, ok := authPayload["team_id"].(float64)
	if !ok || managerTeamID == 0 {
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Verify that the project belongs to the manager's team.
	project, err := server.store.GetProject(ctx, req.ProjectID)
	if err != nil {
		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("project not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if project.TeamID != int64(managerTeamID) {
		err := errors.New("forbidden: you can only create tasks for projects in your own team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}
	// Step 3: Extract required skills from the task description using the skillzProcessor.
	requiredSkills, err := server.skillzProcessor.ExtractAndNormalize(ctx, req.Description)
	if err != nil {
		log.Printf("❌ skillzProcessor error during task creation: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not process task description for skills"})
		return
	}

	// Step 4: Prepare the parameters for the database transaction.
	// Use the ProcessNewTask transaction which handles task and skill creation atomically.
	arg := db.ProcessNewTaskTxParams{
		CreateTaskParams: db.CreateTaskParams{
			ProjectID:  	pgtype.Int8{Int64: req.ProjectID, Valid: true}, 
			Title:       	req.Title,
			Description: 	pgtype.Text{String: req.Description, Valid: true}, 
			Status: 		db.TaskStatusOpen,
			Priority:   	db.TaskPriority(req.Priority), 
		},
		RequiredSkillNames: requiredSkills,
	}

	// Step 5: Execute the transaction.
	result, err := server.store.ProcessNewTask(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Step 6: Return the newly created task and its linked skills.
	ctx.JSON(http.StatusCreated, result)
}
