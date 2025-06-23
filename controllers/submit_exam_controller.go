package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"examination-papers/data/storage"
	"examination-papers/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/joho/godotenv/autoload"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"time"
)

const QUESTIONTASKSQUEUE = "question_tasks_queue"
const STUDENTANSWERSQUEUE = "student_answers_queue"
const SUBMITIDEXAMSUB = "submit_exam:"
const SUBMITIDANSWERSUB = "submit:answer:"

// get env
var HANDLEANSWERAPPID = os.Getenv("HANDLE_ANSWER_APPID")
var HANDLEQUESTIONAPPID = os.Getenv("HANDLE_QUESTION_APPID")
var EXAMPAPERSMATHAPPID = os.Getenv("EXAM_PAPER_MATH_APPID")
var HANDLESCOREAPPID = os.Getenv("HANDLE_SCORE_APPID")

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
	CardID   string `json:"card_id" validate:"required"`
	Callback string `json:"callback" validate:"required"`   // 回调地址
	Items    []Item `json:"items" validate:"required,dive"` // List of questions
}

type Item struct {
	ItemID    string `json:"item_id" validate:"required"`    // Question ID
	Body      string `json:"body" validate:"required"`       // Question body
	Answer    string `json:"answer" validate:"required"`     // Question answer
	FullScore string `json:"full_score" validate:"required"` // Maximum score for the question
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
	ExamID    string `json:"exam_id" validate:"required"`    // Exam ID
	ItemID    string `json:"item_id" validate:"required"`    // Question ID
	Body      string `json:"body" validate:"required"`       // Question body
	Answer    string `json:"answer" validate:"required"`     // Question answer
	SubmitId  string `json:"submit_id" validate:"required"`  // Unique ID for the submission
	CallBack  string `json:"callback" validate:"required"`   // Callback URL for result notification
	FullScore string `json:"full_score" validate:"required"` // Maximum score for the question
}

type ExamStudentAnswerTask struct {
	BlockID   string   `json:"block_id" validate:"required"`        // Unique ID for the answer block
	ExamID    string   `json:"exam_id" validate:"required"`         // Exam ID
	ItemID    string   `json:"item_id" validate:"required"`         // Question ID
	StudentID string   `json:"student_id" validate:"required"`      // Student ID
	Answers   []string `json:"answer" validate:"required,dive,url"` // Student's answer list (image URLs)
	SubmitId  string   `json:"submit_id" validate:"required"`       // Unique ID for the submission
	Callback  string   `json:"callback" validate:"required,url"`    // Callback URL for result notification
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

	if req.CardID == "" || len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    1,
			"message": "Exam ID and items are required",
		})
	}
	ctx := context.Background()
	submitId := uuid.NewString()
	sc.redisClient.Set(ctx, SUBMITIDEXAMSUB+submitId, len(req.Items), 4*time.Hour)
	for _, item := range req.Items {
		task := ExamItemTask{
			ExamID:    req.CardID,
			ItemID:    item.ItemID,
			Body:      item.Body,
			Answer:    item.Answer,
			FullScore: item.FullScore,
			CallBack:  req.Callback,
			SubmitId:  submitId,
		}
		taskBytes, err := json.Marshal(task)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":    1,
				"message": "Failed to serialize task",
			})
		}

		err = sc.redisClient.LPush(ctx, QUESTIONTASKSQUEUE, taskBytes).Err()
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
		result, err := sc.redisClient.BLPop(ctx, 0, QUESTIONTASKSQUEUE).Result()
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
			"answer":     examTask.Answer,
			"full_score": examTask.FullScore,
		}

		// 处理 answer
		answerResp, err := utils.AgentRequest(HANDLEANSWERAPPID, bizParams)
		if err != nil {
			log.Printf("[Worker] AgentRequest error: %v", err)
			continue
		}

		// 调用外部api 处理 body
		bodyParams := map[string]interface{}{
			"question": examTask.Body,
		}
		// 处理 原问题
		bodyResp, err := utils.AgentRequest(HANDLEQUESTIONAPPID, bodyParams)
		if err != nil {
			log.Printf("[Worker] AgentRequest error for body: %v", err)
			continue
		}

		// todo : be upsert
		query := `
		INSERT INTO exam_items (
			exam_id, item_id, body, correct_answer, body_result, correct_answer_result
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (item_id)
		DO UPDATE SET
		exam_id = EXCLUDED.exam_id,
			body = EXCLUDED.body,
			correct_answer = EXCLUDED.correct_answer,
			body_result = EXCLUDED.body_result,
			correct_answer_result = EXCLUDED.correct_answer_result,
			updated_at = NOW()
`
		_, err = sc.db.Exec(query, examTask.ExamID, examTask.ItemID, examTask.Body, examTask.Answer, bodyResp.Text, answerResp.Text)
		remaining, err := sc.redisClient.Decr(ctx, SUBMITIDEXAMSUB+examTask.SubmitId).Result()
		if err != nil {
			log.Printf("[Worker] Failed to insert exam item: %v", err)
			continue
		}

		log.Printf("[Worker] Successfully processed exam item: %s", examTask.ItemID)

		if remaining <= 0 {
			lockKey := "submit_exam:lock:" + examTask.SubmitId
			ok, _ := sc.redisClient.SetNX(ctx, lockKey, "1", 5*time.Second).Result()
			if ok {
				// 执行回调
				sc.notifyExamCallback(examTask)

				sc.redisClient.Del(ctx, SUBMITIDEXAMSUB+examTask.SubmitId)
				sc.redisClient.Del(ctx, lockKey)
			}
		}
	}
}

func (sc *SubmitExamCase) SubmitAnswerController(c *fiber.Ctx) error {
	var req SubmitAnswerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    1,
			"message": "Invalid request body",
		})
	}

	submitId := uuid.NewString()
	sc.redisClient.Set(context.Background(), SUBMITIDANSWERSUB+submitId, len(req.StudentAnswers), 120*time.Second) // 2小时过期

	tx, err := sc.db.Beginx()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "DB error")
	}

	for _, ans := range req.StudentAnswers {
		query := `INSERT INTO exam_blocks 
			(block_id, exam_id, student_id, item_id, answer, callback, status) 
			VALUES ($1, $2, $3, $4, $5, $6, 'pending')`
		_, err := tx.Exec(query, ans.BlockID, req.ExamID, ans.StudentID, ans.ItemID, pq.Array(ans.AnswerList), req.Callback)
		if err != nil {
			tx.Rollback()
			log.Printf("[SubmitAnswerController] Insert failed for student answer: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "Insert failed for student answer")
		}

		// 同步推入 Redis 队列（每道题为单位）
		task := ExamStudentAnswerTask{
			BlockID:   ans.BlockID,
			ExamID:    req.ExamID,
			ItemID:    ans.ItemID,
			StudentID: ans.StudentID,
			Answers:   ans.AnswerList,
			Callback:  req.Callback,
			SubmitId:  submitId,
		}
		payload, _ := json.Marshal(task)
		sc.redisClient.RPush(context.Background(), STUDENTANSWERSQUEUE, payload)
	}

	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Commit failed")
	}

	return c.JSON(fiber.Map{
		"code":    0,
		"message": "Submitted successfully",
	})
}

func (sc *SubmitExamCase) SubmitAnswerWorker() {
	for {
		ctx := context.Background()
		result, err := sc.redisClient.BLPop(ctx, 0, STUDENTANSWERSQUEUE).Result()
		if err != nil {
			log.Printf("[SubmitAnswerWorker] Redis BLPop error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		// result[0] 是 queue name，result[1] 是数据
		if len(result) < 2 {
			continue
		}
		data := result[1]
		var task ExamStudentAnswerTask
		if err := json.Unmarshal([]byte(data), &task); err != nil {
			log.Printf("[SubmitAnswerWorker] JSON decode failed: %v", err)
			continue
		}
		log.Printf("[SubmitAnswerWorker] Processing answer for block: %s, exam: %s, student: %s", task.BlockID, task.ExamID, task.StudentID)
		// 根据 ItemID 获取题目详情
		query := `SELECT body_result, correct_answer_result FROM exam_items WHERE item_id = $1`
		var bodyResult, correctAnswerResult string
		err = sc.db.QueryRow(query, task.ItemID).Scan(&bodyResult, &correctAnswerResult)
		if err != nil {
			log.Printf("[SubmitAnswerWorker] Failed to fetch item details: %v", err)
			continue
		}
		// 调用判卷方法
		bizParams := map[string]interface{}{
			"studentAnswer": task.Answers[0], // todo : 目前数学只处理第一个答案
			// "body":          bodyResult,
			"correctAnswer": correctAnswerResult,
		}
		// 批卷子
		taskResult, err := utils.AgentRequest(EXAMPAPERSMATHAPPID, bizParams)
		if err != nil {
			log.Printf("[SubmitAnswerWorker] AgentRequest error: %v", err)
			continue
		}
		log.Printf("Process successful!")
		// 更新数据库
		updateQuery := `UPDATE exam_blocks SET status = 'completed', result = $1 WHERE block_id = $2`
		_, err = sc.db.Exec(updateQuery, taskResult.Text, task.BlockID)
		if err != nil {
			log.Printf("[SubmitAnswerWorker] Failed to update exam block: %v", err)
			continue
		}
		// 请求百炼智能体 分析分数
		scoreRes, err := utils.AgentRequest(HANDLESCOREAPPID, map[string]interface{}{
			"res": taskResult.Text,
		})

		if err != nil {
			log.Printf("[SubmitAnswerWorker] AgentRequest for score error: %v", err)
			continue
		}
		var scoreResult struct {
			FullScore string `json:"full_score"`
			Score     string `json:"score"`
		}
		err = json.Unmarshal([]byte(scoreRes.Text), &scoreResult)
		if err != nil {
			log.Printf("[SubmitAnswerWorker] JSON unmarshal for score error: %v", err)
			continue
		}

		resultItem := map[string]string{
			"block_id":   task.BlockID,
			"item_id":    task.ItemID,
			"student_id": task.StudentID,
			"result":     taskResult.Text,
			"score":      scoreResult.Score,
			"full_score": scoreResult.FullScore,
		}
		resultJSON, _ := json.Marshal(resultItem)
		sc.redisClient.RPush(ctx, "submit:result:"+task.SubmitId, resultJSON)

		submitKey := SUBMITIDANSWERSUB + task.SubmitId
		remaining, err := sc.redisClient.Decr(context.Background(), submitKey).Result()
		if err != nil {
			log.Printf("[SubmitAnswerWorker] Redis DECR error: %v", err)
			continue
		}

		if remaining <= 0 {
			lockKey := "submit:lock:" + task.SubmitId
			ok, _ := sc.redisClient.SetNX(context.Background(), lockKey, "1", 5*time.Second).Result()
			if ok {
				resultKey := "submit:result:" + task.SubmitId
				resultList, err := sc.redisClient.LRange(context.Background(), "submit:result:"+task.SubmitId, 0, -1).Result()
				if err != nil {
					log.Printf("[SubmitAnswerWorker] Redis LRange error: %v", err)
					return
				}

				sc.notifyCallback(task, resultList)

				sc.redisClient.Del(context.Background(), SUBMITIDANSWERSUB+task.SubmitId)
				sc.redisClient.Del(context.Background(), resultKey)
				sc.redisClient.Del(context.Background(), lockKey)
			}
		}
	}
}

type CallbackPayload struct {
	ExamID        string          `json:"exam_id"`
	StudentResult []StudentResult `json:"student_result"`
}

type StudentResult struct {
	StudentID string       `json:"student_id"`
	ItemID    string       `json:"item_id"`
	BlockID   string       `json:"block_id"`
	Result    ResultDetail `json:"result"`
}

type ResultDetail struct {
	OverAllFeedBack string `json:"over_all_feed_back"`
	Score           string `json:"score"`
	FullScore       string `json:"full_score"`
	Time            string `json:"time"`
}

func (sc *SubmitExamCase) notifyCallback(task ExamStudentAnswerTask, resultList []string) {
	var studentResults []StudentResult
	for _, res := range resultList {
		var resMap map[string]interface{}
		err := json.Unmarshal([]byte(res), &resMap)
		if err != nil {
			log.Printf("[notifyCallback] JSON unmarshal error: %v", err)
			continue
		}
		studentResults = append(studentResults, StudentResult{
			StudentID: task.StudentID,
			ItemID:    task.ItemID,
			BlockID:   task.BlockID,
			Result: ResultDetail{
				Score:           resMap["score"].(string),
				FullScore:       resMap["full_score"].(string),
				OverAllFeedBack: resMap["result"].(string),
				Time:            time.Now().Format(time.RFC3339), // 使用当前时间作为时间戳
			},
		})
	}
	payload := CallbackPayload{
		ExamID:        task.ExamID,
		StudentResult: studentResults,
	}
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[notifyCallback] JSON marshal error: %v", err)
		return
	}
	log.Printf(string(payloadJson))
	// 发送 HTTP POST 请求
	resp, err := http.Post(task.Callback, "application/json", bytes.NewReader(payloadJson))
	if err != nil {
		log.Printf("[notifyCallback] POST request error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[notifyCallback] Callback returned non-200 status: %d", resp.StatusCode)
	}
}

func (sc *SubmitExamCase) notifyExamCallback(task ExamItemTask) {
	payload := map[string]interface{}{
		"card_id": task.ExamID,
		"result":  "Done",
	}
	payloadJson, _ := json.Marshal(payload)
	resp, err := http.Post(task.CallBack, "application/json", bytes.NewReader(payloadJson))
	if err != nil {
		log.Printf("[notifyExamCallback] POST request error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[notifyExamCallback] Callback returned non-200 status: %d", resp.StatusCode)
	}
}
