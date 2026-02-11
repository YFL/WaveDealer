package impl

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type Server struct {
}

func (s *Server) RequestYoutubeSong(c *gin.Context) {
	fmt.Println("kurva anyad")
}
