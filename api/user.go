package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type createUserRequest struct {
	Name         string
	Email        string
	TeamID       int8
	PasswordHash string
	Role         string
}

func (server *Server) createUser(ctx *gin.Context) {

}
