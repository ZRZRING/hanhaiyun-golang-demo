package main

import (
	"examination-papers/configs"
	"examination-papers/controllers"
	"examination-papers/data/db"
	"examination-papers/data/redis"
	"examination-papers/middleware"
	"examination-papers/routes"
	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
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
	redisConfig := redis.Config{
		Host:     "localhost",
		Port:     6379,
		Password: "eYVX7EwVmmxKPCDmwMtyKVge8oLd2t81",
		DB:       0,
	}
	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		panic(err)
	}
	examCase := controllers.NewSubmitExamCase(dbClient.DB, nil, redisClient.Client)
	for i := 0; i < 7; i++ { // 启动 5 个 worker
		go examCase.SubmitExamWorker()
	}
	for i := 0; i < 5; i++ {
		go examCase.SubmitAnswerWorker()
	}
	routes.PublicRoutes(app, examCase)
	log.Printf("Starting server on port %s", os.Getenv("PORT"))
	port := os.Getenv("PORT")
	app.Listen(port)
}
