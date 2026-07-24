package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	wallet "github.com/Lirikman/rest-wallet/api"
	generated "github.com/Lirikman/rest-wallet/db/generated"
	router "github.com/Lirikman/rest-wallet/router"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	// загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found, reading system variables")
	}
	host := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	pathMigrations := os.Getenv("PATH_MIGRATIONS")

	// формируем строку подключения к локальной базе данных
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, dbPort, dbName)

	// применение миграций
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("connection error: %v", err)
	}
	// освобождаем ресурсы
	defer func() {
		if closeErr := db.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("dialect installation error: %v", err)
	}
	if err := goose.Up(db, pathMigrations); err != nil {
		log.Fatalf("error applying migrations: %v", err)
	}
	log.Println("migrations have been successfully applied")

	// инициализация пула соединений
	conn, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v\n", err)
	}
	defer conn.Close()

	queries := generated.New(conn)

	// создаём маршрутизатор
	r := router.SetupRouter()

	// регистрируем маршруты
	r.GET("/api/v1/wallet", wallet.ListWallets(queries))
	r.GET("/api/v1/wallet/by-id/:id", wallet.GetWalletFromId(queries))
	r.GET("/api/v1/wallet/:wallet_id", wallet.GetBalanceFromWalletId(queries))
	r.POST("/api/v1/wallet", wallet.CreateWallet(queries))
	r.PUT("/api/v1/wallet/:id", wallet.UpdateWallet(queries))
	r.DELETE("/api/v1/wallet/:id", wallet.DeleteWallet(queries))

	// запускаем сервер на порту 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server startup error")
	}
}
