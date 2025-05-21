package main

import (
	"examination-papers/configs"
	"examination-papers/controllers"
	"examination-papers/data/db"
	"examination-papers/middleware"
	"examination-papers/routes"
	"github.com/gofiber/fiber/v2"
)

func main() {
	//app := fiber.New()
	//
	//app.Get("/", func(c *fiber.Ctx) error {
	//	return c.SendString("Hello, World!")
	//})
	//
	//app.Listen(":3000")
	dbConfig := db.Config{
		Host:     "localhost",
		Port:     25432,
		User:     "postgres",
		Password: "postgres",
		DBName:   "postgres",
		SSLMode:  "disable",
	}
	config := configs.FiberConfig()
	app := fiber.New(config)
	middleware.FiberMiddleware(app)
	dbClient, err := db.NewPostgresClient(dbConfig)
	if err != nil {
		panic(err)
	}
	examCase := controllers.NewSubmitExamCase(dbClient.DB, nil)
	routes.PublicRoutes(app, examCase)

	app.Listen(":3000")
}
