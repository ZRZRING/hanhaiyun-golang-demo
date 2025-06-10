package routes

import (
	"examination-papers/controllers"
	"github.com/gofiber/fiber/v2"
)

// PublicRoutes func for describe group of public routes.
func PublicRoutes(a *fiber.App, sc *controllers.SubmitExamCase) {
	// Create routes group.
	route := a.Group("/api/v1")

	// Routes for GET method:
	// route.Get("/books", controllers.GetBooks)   // get list of all books
	route.Post("/submit_exam", sc.SubmitExamController)
	route.Post("/submit_student_answer", sc.SubmitAnswerController)
}
