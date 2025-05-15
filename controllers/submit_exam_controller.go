package controllers

import (
	"examination-papers/data/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

type SubmitExamCase struct {
	db          *sqlx.DB
	minioClient *storage.MinioClient
}

func NewSubmitExamCase(db *sqlx.DB, minioClient *storage.MinioClient) *SubmitExamCase {
	return &SubmitExamCase{
		db:          db,
		minioClient: minioClient,
	}
}

type SubmitRequest struct {
	ExamID string `json:"exam_id" validate:"required"`
	Items  []Item `json:"items" validate:"required,dive"` // List of questions
}

type Item struct {
	ItemID string `json:"item_id" validate:"required"` // Question ID
	Body   string `json:"body" validate:"required"`    // Question body
	Answer string `json:"answer" validate:"required"`  // Question answer
}

func (sc *SubmitExamCase) SubmitExamController(c *fiber.Ctx) error {
	var req SubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    1,
			"message": "Invalid request body",
		})
	}

	if req.ExamID == "" || len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    1,
			"message": "Exam ID and items are required",
		})
	}

	// todo : process

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "Exam submitted successfully",
	})
}
