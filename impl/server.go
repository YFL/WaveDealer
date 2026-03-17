package impl

import (
	"fmt"
	"net/http"

	"github.com/YFL/WaveDealer/api"
	wd "github.com/YFL/WaveDealer/app"
	"github.com/gin-gonic/gin"
)

type Server struct {
	app *wd.WaveDealer
}

type ServerOption func(s *Server)

func WitApp(app *wd.WaveDealer) ServerOption {
	return func(s *Server) {
		s.app = app
	}
}

func NewServer(opts ...ServerOption) *Server {
	s := &Server{}
	for _, o := range opts {
		o(s)
	}

	return s
}

func (s *Server) RequestYoutubeSong(c *gin.Context) {
	var bodyJson api.RequestYoutubeSongJSONRequestBody
	if err := c.BindJSON(&bodyJson); err != nil {
		errStr := fmt.Sprintf("Couldn't bind the request body to api.RequestYoutubeSongJSONRequestBody: %s\n", err)
		c.JSON(http.StatusInternalServerError, errStr)
		fmt.Println(errStr)
		return
	}

	s.app.RequestYoutubeSong(bodyJson.Url)
	c.JSON(http.StatusOK, bodyJson.Url)
}
