package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/pranav244872/synapse/db/sqlc"
)

// Server serves HTTP requests
type Server struct {
	store *db.Store
	router *gin.Engine
}

func NewServer(store *db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	// add routes to router

	router.POST("/auth/register", server.createUser)

	server.router = router
	return server
}


