package main

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"platform/functions/internal/db"
	"platform/functions/internal/domain/function"
	"platform/functions/internal/domain/job"
	"platform/functions/internal/executor"
	httpTransport "platform/functions/internal/transport/http"
)

func main() {
	dsn := os.Getenv("FUNCTION_DB_DSN")
	if dsn == "" {
		log.Fatal("FUNCTION_DB_DSN not set")
	}

	dbConn, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("failed to connect DB: %v", err)
	}
	defer dbConn.Close()

	if err := db.RunMigrations(dbConn); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	funcRepo := function.NewPostgresRepository(dbConn)
	jobRepo := job.NewPostgresRepository(dbConn)

	dockerRunner, err := executor.NewDockerRunner()
	if err != nil {
		log.Fatalf("failed to init DockerRunner: %v", err)
	}

	execSvc := executor.NewExecutor(jobRepo, funcRepo, dockerRunner, 5)
	execSvc.Start()
	defer execSvc.Stop()

	r := gin.Default()

	httpTransport.SetupRoutes(r, funcRepo, jobRepo, execSvc)

	log.Println("[Function-Service] listening on :8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatal(err)
	}
}

func connectDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
