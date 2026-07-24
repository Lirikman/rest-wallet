run: # сборка контейнера и запуск api
	docker compose --env-file config.env up --build -d

stop: # оастновка и удаление контейнера
	docker compose down

lint: # проверка кода линтером golangci-lint
	golangci-lint run
	
test: # запуск тестов
	go test -v ./...