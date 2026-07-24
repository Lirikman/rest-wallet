.PHONY: run stop

run: # сборка контейнера и запуск в фоновом режиме
	docker compose --env-file config.env up --build -d

stop: # безопасная остановка с запасом времени в 30 секунд
	docker compose down -t 30

lint: # проверка кода линтером golangci-lint
	golangci-lint run
	
test: # запуск тестов
	go test -v ./...