package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Book struct {
	ID     uint   `form:"-"`
	Title  string `form:"title" binding:"required"`
	Author string `form:"author" binding:"required"`
}

func setupDatabase(db *gorm.DB) error {
	err := db.AutoMigrate(
		&Book{},
	)
	if err != nil {
		return fmt.Errorf("Error migrating database: %s", err)
	}
	return nil
}

func connectDatabase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("database", db)
	}
}

func bookIndexHandler(c *gin.Context) {
	db := c.Value("database").(*gorm.DB)
	books := []Book{}
	if err := db.Find(&books).Error; err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "books/index.html", gin.H{"books": books})
}

func bookNewGetHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "books/new.html", gin.H{})
}

func bookNewPostHandler(c *gin.Context) {
	book := &Book{}
	if err := c.ShouldBind(book); err != nil {
		verrs := err.(validator.ValidationErrors)
		messages := make([]string, len(verrs))

		for i, verr := range verrs {
			messages[i] = fmt.Sprintf(
				"%s is required, but was empty.",
				verr.Field())
		}

		c.HTML(http.StatusBadRequest, "books/new.html", gin.H{"errors": messages})
		return
	}

	db := c.Value("database").(*gorm.DB)
	if err := db.Create(&book).Error; err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Redirect(http.StatusFound, "/books/")
}

func defaultHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "default.html", gin.H{})
}

func setupRouter(router *gin.Engine, db *gorm.DB) {
	router.LoadHTMLGlob("templates/**/*.html")
	router.Use(connectDatabase(db))
	router.GET("/books/", bookIndexHandler)
	router.GET("/books/new", bookNewGetHandler)
	router.POST("/books/new", bookNewPostHandler)
	router.Static("/static", "./static/")
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/books/")
	})
}

func main() {

	db, err := gorm.Open(sqlite.Open("aklatan.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}
	err = setupDatabase(db)
	if err != nil {
		log.Fatalf("Database setup error: %s", err)
	}

	router := gin.Default()
	setupRouter(router, db)
	err = router.Run(":3000")
	if err != nil {
		log.Fatalf("gin Run error: %s", err)
	}
}
