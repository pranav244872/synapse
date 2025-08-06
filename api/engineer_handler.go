// api/engineer_handler.go

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
// Engineer Task Handlers
////////////////////////////////////////////////////////////////////////

// getCurrentTask retrieves the single task currently assigned and in-progress for the engineer.
func (server *Server) getCurrentTask(ctx *gin.Context) {
	log.Printf("DEBUG: Starting getCurrentTask handler")

	// Extract user authentication information from request context
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}
	
	// Convert user ID from token payload to int64 for database queries
	engineerID := int64(authPayload["user_id"].(float64))

	// Query database for engineer's currently active task
	task, err := server.store.GetCurrentTaskForEngineer(ctx, pgtype.Int8{Int64: engineerID, Valid: true})
	if err != nil {
		// Handle case where engineer has no active tasks
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("DEBUG: No active task found for engineer %d", engineerID)
			ctx.JSON(http.StatusNoContent, nil) // Return 204 No Content as requested
			return
		}
		// Handle database or other system errors
		log.Printf("ERROR: Failed to get current task for engineer %d: %v", engineerID, err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Return the active task details to the engineer
	ctx.JSON(http.StatusOK, task)
}

// getTaskDetails retrieves full, rich details for any single task, as long as it belongs to the engineer's team.
func (server *Server) getTaskDetails(ctx *gin.Context) {
	log.Printf("DEBUG: Starting getTaskDetails handler")

	// Parse task ID from URL path parameters
	var uriReq struct {
		ID int64 `uri:"id" binding:"required,min=1"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Extract team ID from engineer's authentication token
	authPayload, _ := getAuthorizationPayload(ctx)
	teamID := int64(authPayload["team_id"].(float64))

	// Retrieve task from database to validate existence
	task, err := server.store.GetTask(ctx, uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(errors.New("task not found")))
		return
	}

	// Get project information to verify team ownership
	project, err := server.store.GetProject(ctx, task.ProjectID.Int64)
	if err != nil || project.TeamID != teamID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("you do not have permission to view this task")))
		return
	}

	// Fetch comprehensive task details including project information
	taskDetails, err := server.store.GetTaskDetailsWithProject(ctx, uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Retrieve skills required for this specific task
	requiredSkills, err := server.store.GetSkillsForTask(ctx, uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Transform skills data into client-friendly response format
	type skillResponse struct {
		ID        int64  `json:"id"`
		SkillName string `json:"skillName"`
	}
	skillsRsp := make([]skillResponse, len(requiredSkills))
	for i, s := range requiredSkills {
		skillsRsp[i] = skillResponse{ID: s.ID, SkillName: s.SkillName}
	}

	// Construct comprehensive task response with all relevant details
	response := gin.H{
		"id":             taskDetails.ID,
		"title":          taskDetails.Title,
		"description":    taskDetails.Description.String,
		"projectName":    taskDetails.ProjectName,
		"requiredSkills": skillsRsp,
		"activityLog":    []string{}, // Return empty log for now as planned
	}

	ctx.JSON(http.StatusOK, response)
}

// completeTask marks the engineer's currently assigned task as 'done'.
func (server *Server) completeTask(ctx *gin.Context) {
	log.Printf("DEBUG: Starting completeTask handler")

	// Parse task ID from URL path parameters
	var uriReq struct {
		ID int64 `uri:"id" binding:"required,min=1"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Extract engineer ID from authentication token
	authPayload, _ := getAuthorizationPayload(ctx)
	engineerID := int64(authPayload["user_id"].(float64))

	// Retrieve task to validate assignment and ownership
	taskToComplete, err := server.store.GetTask(ctx, uriReq.ID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, errorResponse(errors.New("task not found")))
		return
	}

	// Verify that the requesting engineer is actually assigned to this task
	if !taskToComplete.AssigneeID.Valid || taskToComplete.AssigneeID.Int64 != engineerID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("you can only complete tasks assigned to you")))
		return
	}

	// Execute task completion transaction (updates task status and engineer availability)
	result, err := server.store.CompleteTaskTx(ctx, db.CompleteTaskTxParams{TaskID: uriReq.ID})
	if err != nil {
		log.Printf("ERROR: Failed to complete task %d: %v", uriReq.ID, err)
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Log successful completion and return updated task data
	log.Printf("DEBUG: Engineer %d completed task %d", engineerID, uriReq.ID)
	ctx.JSON(http.StatusOK, result.CompletedTask)
}

////////////////////////////////////////////////////////////////////////
// Engineer Project & History Handlers
////////////////////////////////////////////////////////////////////////

// listProjectTasksForEngineer retrieves a read-only list of all tasks for a specific project.
func (server *Server) listProjectTasksForEngineer(ctx *gin.Context) {
	log.Printf("DEBUG: Starting listProjectTasksForEngineer handler")

	// Parse project ID from URL path parameters
	var uriReq struct {
		ID int64 `uri:"id" binding:"required,min=1"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Extract team ID from engineer's authentication token
	authPayload, _ := getAuthorizationPayload(ctx)
	teamID := int64(authPayload["team_id"].(float64))

	// Retrieve project information to validate existence and team membership
	project, err := server.store.GetProject(ctx, uriReq.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("project not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Verify engineer belongs to the same team as the project
	if project.TeamID != teamID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("you do not have permission to view tasks for this project")))
		return
	}

	// Fetch all tasks for the project with assignee information
	tasks, err := server.store.ListTasksWithAssigneeNames(ctx, db.ListTasksWithAssigneeNamesParams{
		ProjectID: pgtype.Int8{Int64: project.ID, Valid: true}, // Convert int64 to pgtype.Int8 for database query
		Limit:     500, // High limit to get all tasks
		Offset:    0,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Return complete list of project tasks to engineer
	ctx.JSON(http.StatusOK, tasks)
}

// getTaskHistory retrieves a paginated list of the engineer's completed tasks.
func (server *Server) getTaskHistory(ctx *gin.Context) {
	log.Printf("DEBUG: Starting getTaskHistory handler")

	// Parse pagination and search parameters from query string
	var queryReq struct {
		PageID   int32  `form:"page_id" binding:"required,min=1"`
		PageSize int32  `form:"page_size" binding:"required,min=5,max=50"`
		Search   string `form:"search"` // Optional
	}
	if err := ctx.ShouldBindQuery(&queryReq); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Extract engineer ID from authentication token
	authPayload, _ := getAuthorizationPayload(ctx)
	engineerID := int64(authPayload["user_id"].(float64))

	// Prepare search query with wildcard pattern for database ILIKE operation
	searchQuery := "%"
	if queryReq.Search != "" {
		searchQuery = "%" + queryReq.Search + "%"
	}

	// Query paginated task history for the engineer with optional search filtering
	history, err := server.store.GetEngineerTaskHistory(ctx, db.GetEngineerTaskHistoryParams{
		AssigneeID: pgtype.Int8{Int64: engineerID, Valid: true}, // Convert engineer ID to pgtype.Int8
		Limit:      queryReq.PageSize,
		Offset:     (queryReq.PageID - 1) * queryReq.PageSize,
		Search:     searchQuery, // Pass search pattern directly as string
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Get total count of matching tasks for pagination metadata
	totalCount, err := server.store.GetEngineerTaskHistoryCount(ctx, db.GetEngineerTaskHistoryCountParams{
		AssigneeID: pgtype.Int8{Int64: engineerID, Valid: true}, // Convert engineer ID to pgtype.Int8
		Search:     searchQuery, // Pass search pattern directly as string
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Construct paginated response with history data and total count
	response := paginatedResponse[db.GetEngineerTaskHistoryRow]{
		TotalCount: totalCount,
		Data:       history,
	}

	ctx.JSON(http.StatusOK, response)
}
