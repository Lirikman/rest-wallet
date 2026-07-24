package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	wallet "github.com/Lirikman/rest-wallet/api"
	generated "github.com/Lirikman/rest-wallet/db/generated"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

// создание маршрутизатора Gin
func setupRouter() *gin.Engine {
	router := gin.Default()
	router.ForwardedByClientIP = true
	// настраиваем доверенные прокси
	proxies := []string{"127.0.0.1", "::1"}
	err := router.SetTrustedProxies(proxies)
	if err != nil {
		log.Fatalf("error while setting up proxy")
	}
	// настройка политики разрешений
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://localhost:8080/"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Type"}
	router.Use(cors.New(config))
	// подключаем инструмент восстановления сбоев
	router.Use(gin.Recovery())
	// подключаем логгер
	router.Use(gin.Logger())
	return router
}

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
	r := setupRouter()

	// регистрируем маршруты
	r.GET("/api/v1/wallets", wallet.ListWallets(queries))
	r.GET("/api/v1/wallets/:id", wallet.GetWalletFromWalletId(queries))
	r.GET("/api/v1/wallets/:wallet_uuid", wallet.GetWalletFromWalletId(queries))
	r.POST("/api/v1/wallets", wallet.CreateWallet(queries))
	r.PUT("/api/v1/wallets/:id", wallet.UpdateWallet(queries))
	r.DELETE("/api/v1/wallets/:id", wallet.DeleteWallet(queries))

	// запускаем сервер на порту 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server startup error")
	}
}
