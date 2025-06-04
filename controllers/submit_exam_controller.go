package controllers

import (
	"context"
	"encoding/json"
	"examination-papers/data/storage"
	"examination-papers/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

const TASKSQUEUE = "tasks_queue"

type SubmitExamCase struct {
	db          *sqlx.DB
	minioClient *storage.MinioClient
	redisClient *redis.Client
}

func NewSubmitExamCase(db *sqlx.DB, minioClient *storage.MinioClient, redisClient *redis.Client) *SubmitExamCase {
	return &SubmitExamCase{
		db:          db,
		minioClient: minioClient,
		redisClient: redisClient,
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

type ExamItemTask struct {
	ExamID string `json:"exam_id" validate:"required"` // Exam ID
	ItemID string `json:"item_id" validate:"required"` // Question ID
	Body   string `json:"body" validate:"required"`    // Question body
	Answer string `json:"answer" validate:"required"`  // Question answer
}

func (sc *SubmitExamCase) SubmitExamController(c *fiber.Ctx) error {
	var req SubmitExamRequest
	log.Printf("[SubmitExamController] Received request: %s", c.Body())
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

	for _, item := range req.Items {
		task := ExamItemTask{
			ExamID: req.ExamID,
			ItemID: item.ItemID,
			Body:   item.Body,
			Answer: item.Answer,
		}
		taskBytes, err := json.Marshal(task)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    1,
				"message": "Failed to serialize task",
			})
		}
		ctx := context.Background()
		err = sc.redisClient.LPush(ctx, TASKSQUEUE, taskBytes).Err()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    1,
				"message": "Failed to add task to queue",
			})
		}
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "Task submitted successfully",
	})
}

func (sc *SubmitExamCase) SubmitExamWorker() {
	for {
		ctx := context.Background()
		result, err := sc.redisClient.BLPop(ctx, 0, TASKSQUEUE).Result()
		if err != nil {
			log.Printf("[Worker] Redis BRPop error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		// result[0] 是 queue name，result[1] 是数据
		if len(result) < 2 {
			continue
		}

		data := result[1]
		var examTask ExamItemTask
		if err := json.Unmarshal([]byte(data), &examTask); err != nil {
			log.Printf("[Worker] JSON decode failed: %v", err)
			continue
		}
		log.Printf("[Worker] Processing exam: %s, items: %s", examTask.ExamID, examTask.ItemID)

		// 构造调用参数
		bizParams := map[string]interface{}{
			"answer": examTask.Answer,
		}

		// ⚡ 调用外部 API（AgentRequest）
		resp, err := utils.AgentRequest("3ac741b8e2a34451b8527b407bf289ac", bizParams)
		if err != nil {
			log.Printf("[Worker] AgentRequest error: %v", err)
			continue
		}

		// todo : be upsert
		query := `INSERT INTO exam_items (exam_id, item_id, body, correct_answer, body_result, correct_answer_result) VALUES ($1, $2, $3, $4, $5, $6)`
		_, err = sc.db.Exec(query, examTask.ExamID, examTask.ItemID, examTask.Body, examTask.Answer, "", resp.Text)
		if err != nil {
			log.Printf("[Worker] Failed to insert exam item: %v", err)
			continue
		}
	}
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
	// insert task in db
	taskId := uuid.New()
	for _, answer := range req.StudentAnswers {
		query := `INSERT INTO tasks (task_id, exam_id, block_id, student_id, item_id, answer) VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := tx.Exec(query, taskId, req.ExamID, answer.BlockID, answer.StudentID, answer.ItemID, answer.AnswerList)
		if err != nil {
			log.Printf("Failed to insert task: %v", err)
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    1,
				"message": "Failed to insert task",
			})
		}
	}
	err = tx.Commit()
	return c.JSON(fiber.Map{
		"code":    0,
		"message": "Tasks submitted successfully",
	})
	//
	//for _, answer := range req.StudentAnswers {
	//
	//}
	// todo : need implement
}

func (sc *SubmitExamCase) notifyCallback(callbackURL, blockID string, result *utils.AgentResult) {
	// todo : need implement
}

func (sc *SubmitExamCase) processAnswersAsync(req SubmitAnswerRequest) {
	//go func() {
	//	for _, answer := range req.StudentAnswers {
	//		// Process each answer asynchronously
	//		// get itemUrl by itemId
	//
	//		result, err := utils.AgentMathScore(itemurl, answer.AnswerList[0])
	//		if err != nil {
	//			log.Printf("Error processing answer for block %s: %v", answer.BlockID, err)
	//			continue
	//		}
	//
	//		// Notify callback URL with the result
	//		sc.notifyCallback(req.Callback, answer.BlockID, result)
	//	}
	//}()
}

//// 处理标准答案
//func (sc SubmitExamCase) ProcessStandardAnswer()
