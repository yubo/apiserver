package routes

import (
	"github.com/yubo/apiserver/pkg/server/mux"
	"github.com/yubo/goswagger"
)

type Swagger struct{}

func (p Swagger) Install(c *mux.PathRecorderMux, apidocsPath string) {
	c.HandleFunc("/swagger", redirectTo("/swagger/"))
	c.HandlePrefix("/swagger/", goswagger.New(&goswagger.Config{Name: "apiserver", Url: apidocsPath}).Handler())
}
