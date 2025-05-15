package main

import (
	"examination-papers/configs"
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
	config := configs.FiberConfig()
	app := fiber.New(config)
	middleware.FiberMiddleware(app)

	routes.PublicRoutes(app)

	app.Listen(":3000")
}
