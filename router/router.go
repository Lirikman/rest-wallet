package router

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// создание маршрутизатора Gin
func SetupRouter() *gin.Engine {
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
