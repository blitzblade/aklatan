package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func defaultHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "default.html", gin.H{})
}

func setupRouter(router *gin.Engine) {
	router.LoadHTMLGlob("templates/**/*.html")
	router.GET("/", defaultHandler)
}

func main() {
	router := gin.Default()
	setupRouter(router)
	err := router.Run(":3000")
	if err != nil {
		log.Fatalf("gin Run error: %s", err)
	}
}
