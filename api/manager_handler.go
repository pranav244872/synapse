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
// Invitation Handler (for Managers)
////////////////////////////////////////////////////////////////////////

type inviteEngineerRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func (server *Server) inviteEngineer(ctx *gin.Context) {
	var req inviteEngineerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	authPayload, _ := getAuthorizationPayload(ctx)
	inviterID := int64(authPayload["user_id"].(float64))
	arg := db.CreateInvitationTxParams{
		InviterID:     inviterID,
		EmailToInvite: req.Email,
		RoleToInvite:  db.UserRoleEngineer,
	}
	result, err := server.store.CreateInvitationTx(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrDuplicateInvitation) {
			ctx.JSON(http.StatusConflict, errorResponse(err))
			return
		}
		if errors.Is(err, db.ErrPermissionDenied) {
			ctx.JSON(http.StatusForbidden, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusCreated, result.Invitation)
}

type listSentInvitationsRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=20"`
}

func (server *Server) listSentInvitations(ctx *gin.Context) {
	var req listSentInvitationsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload, _ := getAuthorizationPayload(ctx)
	inviterID := int64(authPayload["user_id"].(float64))

	invitations, err := server.store.ListInvitationsByInviter(ctx, db.ListInvitationsByInviterParams{
		InviterID: inviterID,
		Limit:     req.PageSize,
		Offset:    (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	totalCount, err := server.store.CountInvitationsByInviter(ctx, inviterID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Convert to the unified response struct for API consistency.
	finalInvitations := make([]invitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		finalInvitations = append(finalInvitations, invitationResponse{
			ID:           inv.ID,
			Email:        inv.Email,
			RoleToInvite: inv.RoleToInvite,
			Status:       inv.Status,
			InviterName:  inv.InviterName,
			InviterRole:  inv.InviterRole,
			CreatedAt:    inv.CreatedAt,
		})
	}

	rsp := paginatedResponse[invitationResponse]{
		TotalCount: totalCount,
		Data:       finalInvitations,
	}
	ctx.JSON(http.StatusOK, rsp)
}

////////////////////////////////////////////////////////////////////////
// Project Handler (for Managers)
////////////////////////////////////////////////////////////////////////

type createProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

func (server *Server) createProject(ctx *gin.Context) {
	var req createProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	authPayload, _ := getAuthorizationPayload(ctx)
	teamID, ok := authPayload["team_id"].(float64)
	if !ok || teamID == 0 {
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	arg := db.CreateProjectParams{
		ProjectName: req.Name,
		Description: pgtype.Text{String: req.Description, Valid: true},
		TeamID:      int64(teamID),
	}

	project, err := server.store.CreateProject(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, project)
}

////////////////////////////////////////////////////////////////////////
// Task Handler (for Managers)
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

////////////////////////////////////////////////////////////////////////
// ## Recommendation Handler (for Managers)
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

	task, err := server.store.GetTask(ctx, req.TaskID)
	if err != nil {
		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("task not found")))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	project, err := server.store.GetProject(ctx, task.ProjectID.Int64)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if project.TeamID != int64(managerTeamID) {
		err := errors.New("forbidden: this task does not belong to your team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	requiredSkills, err := server.store.GetSkillsForTask(ctx, req.TaskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if len(requiredSkills) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"recommendations": []EnrichedRecommendation{}})
		return
	}

	var skillIDs []int32
	for _, skill := range requiredSkills {
		skillIDs = append(skillIDs, int32(skill.ID))
	}

	limit := 10
	if req.Limit > 0 && req.Limit <= 50 {
		limit = req.Limit
	}
	recommenderReqPayload := recommenderAPIRequest{SkillIDs: skillIDs, Limit: limit}
	recommenderBody, _ := json.Marshal(recommenderReqPayload)
	request, _ := http.NewRequest("POST", server.config.RecommenderAPIURL, bytes.NewBuffer(recommenderBody))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Internal-API-Key", server.config.RecommenderAPIKey)

	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, errorResponse(errors.New("recommendation service is unavailable")))
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		errText := fmt.Sprintf("recommendation service failed: %s", string(bodyBytes))
		ctx.JSON(http.StatusServiceUnavailable, errorResponse(errors.New(errText)))
		return
	}

	var recommenderResp recommenderAPIResponse
	if err := json.NewDecoder(response.Body).Decode(&recommenderResp); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to parse recommendation response")))
		return
	}

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
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"recommendations": enrichedRecommendations})
}
