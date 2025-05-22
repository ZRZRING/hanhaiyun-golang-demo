package controllers

import (
	"examination-papers/data/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"log"
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

type SubmitExamRequest struct {
	ExamID string `json:"exam_id" validate:"required"`
	Items  []Item `json:"items" validate:"required,dive"` // List of questions
}

type Item struct {
	ItemID string `json:"item_id" validate:"required"` // Question ID
	Body   string `json:"body" validate:"required"`    // Question body
	Answer string `json:"answer" validate:"required"`  // Question answer
}

type SubmitAnswerRequest struct {
	ExamID         string          `json:"exam_id" validate:"required"`      // 考试ID
	Callback       string          `json:"callback" validate:"required,url"` // 回调地址
	StudentAnswers []StudentAnswer `json:"student_answers" validate:"required,dive"`
}

type StudentAnswer struct {
	BlockID    string   `json:"block_id" validate:"required"`             // 唯一ID
	StudentID  string   `json:"student_id" validate:"required"`           // 学生ID
	ItemID     string   `json:"item_id" validate:"required"`              // 试题ID
	AnswerList []string `json:"answer_list" validate:"required,dive,url"` // 学生作答图片列表
}

func (sc *SubmitExamCase) SubmitExamController(c *fiber.Ctx) error {
	var req SubmitExamRequest
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

	tx, err := sc.db.Beginx()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    1,
			"message": "Failed to begin transaction",
		})
	}

	for _, item := range req.Items {
		query := `INSERT INTO exam_items (exam_id, item_id, body, correct_answer) VALUES ($1, $2, $3, $4)`
		_, err := tx.Exec(query, req.ExamID, item.ItemID, item.Body, item.Answer)
		if err != nil {
			log.Fatalf("Failed to insert exam submission: %v", err)
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    1,
				"message": "Failed to insert exam submission",
			})
		}
	}
	err = tx.Commit()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    1,
			"message": "Failed to commit transaction",
		})
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "Exam submitted successfully",
	})
}

func (sc *SubmitExamCase) SubmitAnswerController(c *fiber.Ctx) error {
	var req SubmitAnswerRequest
	err := c.BodyParser(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    1,
			"message": "Invalid request body",
		})
	}

	tx, err := sc.db.Beginx()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    1,
			"message": "Failed to begin transaction",
		})
	}

	for _, answer := range req.StudentAnswers {
		
	}
}
