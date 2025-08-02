// api/recommendation_handler.go
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/pranav244872/synapse/db/sqlc"
	"github.com/jackc/pgx/v5"
)

// Represents the JSON request body for this endpoint.
type getRecommendationsRequest struct {
	TaskID int64 `json:"task_id" binding:"required,min=1"`
	Limit  int   `json:"limit,omitempty"`
}

// Internal struct to build the request for the Python recommender API.
type recommenderAPIRequest struct {
	SkillIDs []int32 `json:"skill_ids"`
	Limit    int     `json:"limit"`
}

// Internal struct to parse the response from the Python recommender API.
type recommenderAPIResponse struct {
	Recommendations []struct {
		UserID int64   `json:"user_id"`
		Score  float64 `json:"score"`
	} `json:"recommendations"`
}

// The final, enriched recommendation object returned to the client.
type EnrichedRecommendation struct {
	UserID int64   `json:"user_id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Score  float64 `json:"score"`
}

// getRecommendations handles the logic for fetching engineer recommendations.
// It is a protected endpoint for managers.
func (server *Server) getRecommendations(ctx *gin.Context) {
	var req getRecommendationsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Step 1: Authorize the request. Only managers can get recommendations.
	authPayload, err := getAuthorizationPayload(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}
	role, _ := authPayload["role"].(string)
	if role != string(db.UserRoleManager) {
		err := errors.New("forbidden: only managers can request recommendations")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}
	managerTeamID, ok := authPayload["team_id"].(float64)
	if !ok || managerTeamID == 0 {
		err := errors.New("forbidden: manager is not assigned to a team")
		ctx.JSON(http.StatusForbidden, errorResponse(err))
		return
	}

	// Step 2: Verify the task belongs to the manager's team.
	// This ensures a manager cannot get recommendations for another team's tasks.
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

	// Step 3: Fetch required skill IDs for the task from the local DB.
	requiredSkills, err := server.store.GetSkillsForTask(ctx, req.TaskID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	if len(requiredSkills) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"recommendations": []EnrichedRecommendation{}})
		return
	}
	skillIDs := make([]int32, len(requiredSkills))
	for i, skill := range requiredSkills {
		skillIDs[i] = int32(skill.ID)
	}

	// Step 4: Call the external Synapse Recommendation API.
	limit := 10 // Default limit
	if req.Limit > 0 && req.Limit <= 50 {
		limit = req.Limit
	}
	recommenderReqPayload := recommenderAPIRequest{
		SkillIDs: skillIDs,
		Limit:    limit,
	}
	recommenderBody, _ := json.Marshal(recommenderReqPayload)

	// Prepare and send the HTTP request.
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

	// Step 5: Decode the response from the recommender service.
	var recommenderResp recommenderAPIResponse
	if err := json.NewDecoder(response.Body).Decode(&recommenderResp); err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(errors.New("failed to parse recommendation response")))
		return
	}

	// Step 6: Enrich the response with user details from our database.
	var enrichedRecommendations []EnrichedRecommendation
	for _, rec := range recommenderResp.Recommendations {
		user, err := server.store.GetUser(ctx, rec.UserID)
		// Only include users that exist in our DB and are on the same team.
		if err == nil && user.TeamID.Int64 == int64(managerTeamID) {
			enrichedRecommendations = append(enrichedRecommendations, EnrichedRecommendation{
				UserID: user.ID,
				Name:   user.Name.String,
				Email:  user.Email,
				Score:  rec.Score,
			})
		}
	}

	// Step 7: Return the final, enriched list.
	ctx.JSON(http.StatusOK, gin.H{"recommendations": enrichedRecommendations})
}
