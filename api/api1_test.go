package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	generated "github.com/Lirikman/rest-wallet/db/generated"
	setup "github.com/Lirikman/rest-wallet/router"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var db *pgxpool.Pool
var router *gin.Engine

func TestMain(m *testing.M) {
	ctx := context.Background()
	// запуск контейнера PostgreSQL
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "secret",
		},
		WaitingFor: wait.ForExposedPort().WithStartupTimeout(60 * time.Second),
	}
	pgCont, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("failed to start container: %v", err)
	}
	defer func() {
		if err := pgCont.Terminate(ctx); err != nil {
			log.Fatalf("context termination error")
		}
	}()
	// получаем адрес и порт
	mappedPort, _ := pgCont.MappedPort(ctx, "5432")
	host, _ := pgCont.Host(ctx)
	// подключение к БД
	dsn := fmt.Sprintf("postgres://test:secret@%s:%s/testdb?sslmode=disable", host, mappedPort.Port())
	db, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()
	// применение миграций
	_, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS wallet (id BIGSERIAL PRIMARY KEY, walletId UUID UNIQUE NOT NULL, operationType VARCHAR(10) NOT NULL, amount INT NOT NULL, CONSTRAINT chk_oper_type CHECK (operationType IN ('DEPOSIT', 'WITHDRAW')));`)
	if err != nil {
		log.Fatalf("failed to create table subscriptions: %v", err)
	}

	// добавление тестовых данных в таблицу wallet
	_, err = db.Exec(ctx, "INSERT INTO wallet (walletId, operationType, amount) VALUES ('c01b3cbc-273a-437d-81d0-94d2c685759d', 'DEPOSIT', 10000), ('c06a41c2-8063-4540-9558-6edf6ba9eafb', 'WITHDRAW', 50000), ('f7df0cc4-ed1c-40eb-8f37-5afa3efb5b5f', 'DEPOSIT', 550500 ), ('122ba01b-6dcc-43f6-a5a2-42450ba74f6b', 'WITHDRAW', 155900), ('6005fafa-cbcd-425a-9e45-b74ac6926f2b', 'WITHDRAW', 122300), ('40f87cbd-cda1-4108-8a94-18502d4cf167', 'DEPOSIT', 1500600), ('2fb7daea-1685-4575-bce1-74103bbf8a6a', 'DEPOSIT', 900340), ('65ae2def-3477-466f-9834-f4c9dc38f802', 'WITHDRAW', 543200);")
	if err != nil {
		log.Fatalf("error adding data to table subscriptions: %v", err)
	}

	// инициализация Gin
	queries := generated.New(db)
	gin.SetMode(gin.TestMode)
	router = setup.SetupRouter()
	// регистрация маршрутов
	router.GET("/api/v1/wallet", ListWallets(queries))
	router.GET("/api/v1/wallet/by-id/:id", GetWalletFromId(queries))
	router.GET("/api/v1/wallet/:wallet_id", GetBalanceFromWalletId(queries))
	router.POST("/api/v1/wallet", CreateWallet(queries))
	router.PUT("/api/v1/wallet/:id", UpdateWallet(queries))
	router.DELETE("/api/v1/wallet/:id", DeleteWallet(queries))
	os.Exit(m.Run())
}

func TestGetWalletsRight(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 8, len(response))
}

func TestPaginationGetWalletsRight1(t *testing.T) {
	// выполнение запроса
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet?range=[0,2]", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := []map[string]any{{"id": float64(1), "walletid": "c01b3cbc-273a-437d-81d0-94d2c685759d", "operationtype": "DEPOSIT", "amount": float64(10000)}, {"id": float64(2), "walletid": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationtype": "WITHDRAW", "amount": float64(50000)}}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestPaginationGetWalletsRight2(t *testing.T) {
	// выполнение запроса
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet?range=[6,20]", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := []map[string]any{{"id": float64(7), "walletid": "2fb7daea-1685-4575-bce1-74103bbf8a6a", "operationtype": "DEPOSIT", "amount": float64(900340)}, {"id": float64(8), "walletid": "65ae2def-3477-466f-9834-f4c9dc38f802", "operationtype": "WITHDRAW", "amount": float64(543200)}}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestPaginationGetWalletsRigh3(t *testing.T) {
	// выполнение запроса
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet?range=[6,6]", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response []map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := []map[string]any{{"id": float64(7), "walletid": "2fb7daea-1685-4575-bce1-74103bbf8a6a", "operationtype": "DEPOSIT", "amount": float64(900340)}}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestPaginationGetWalletWrong1(t *testing.T) {
	// выполнение запроса
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet?range=[15, 2]", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "range values are specified incorrectly"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestPaginationGetWalletWrong2(t *testing.T) {
	// выполнение запроса
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet?range=[a, 2]", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "range must be specified by two numbers, example: [1,4]"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestPaginationGetWalletWrong3(t *testing.T) {
	// выполнение запроса
	r, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet?range=[a, 2, 6, 7]", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "range must be specified by two numbers, example: [1,4]"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestGetWalletFromIDRight(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet/by-id/2", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"id": float64(2), "walletid": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationtype": "WITHDRAW", "amount": float64(50000)}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestGetWalletFromIDWrong1(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet/by-id/1344", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(404), "message": "no records with this ID were found"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestGetWalletFromIDWrong2(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet/by-id/noname", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "incorrect ID value entered"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestGetBalanceFromWalletIDRight(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet/2fb7daea-1685-4575-bce1-74103bbf8a6a", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response int
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := 900340
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestGetBalanceFromWalletIDWrong1(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet/234-25", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestGetBalanceFromWalletIDWrong2(t *testing.T) {
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet/720c6c03-a3f6-4c4d-9433-b3605a6c1b03", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(404), "message": "wallet with this wallet_id not found"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestCreateWalletRight(t *testing.T) {
	// данные для запроса
	data := map[string]any{"walletId": "dfbd21b7-b27f-41ee-b41b-12078ac1035e", "operationType": "DEPOSIT", "amount": 800230}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"id": float64(9), "walletid": "dfbd21b7-b27f-41ee-b41b-12078ac1035e", "operationtype": "DEPOSIT", "amount": float64(800230)}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestCreateWalletWrong1(t *testing.T) {
	// данные для запроса (некорректный walletId)
	data := map[string]any{"walletId": "dfbd2s", "operationType": "WITHDRAW", "amount": 100500}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestCreateWalletWrong2(t *testing.T) {
	// данные для запроса (некорректный operationType)
	data := map[string]any{"walletId": "dfbd21b7-b27f-41ee-b41b-12078ac1035e", "operationType": "BUY", "amount": 100500}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(500), "message": "internal server error"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestCreateWalletWrong3(t *testing.T) {
	// данные для запроса (отрицательный amount)
	data := map[string]any{"walletId": "dfbd21b7-b27f-41ee-b41b-12078ac1035e", "operationType": "WITHDRAW", "amount": -100300}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "amount can range from zero and above"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestCreateWalletWrong4(t *testing.T) {
	// данные для запроса (некорректный amount)
	data := map[string]any{"walletId": "dfbd21b7-b27f-41ee-b41b-12078ac1035e", "operationType": "WITHDRAW", "amount": "hello"}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "invalid request"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletRight(t *testing.T) {
	// данные для запроса
	data := map[string]any{"walletId": "10249ade-4344-4599-8ec1-68a90c843dbe", "operationType": "WITHDRAW", "amount": 250350}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"id": float64(3), "walletid": "10249ade-4344-4599-8ec1-68a90c843dbe", "operationtype": "WITHDRAW", "amount": float64(250350)}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong1(t *testing.T) {
	// данные для запроса (несуществующий id записи)
	data := map[string]any{"walletId": "6046ba8a-ea0f-43ef-b758-a97af67bf7d4", "operationType": "DEPOSIT", "amount": 500670}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/50", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(404), "message": "no records with this ID were found"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong2(t *testing.T) {
	// данные для запроса (некорректный id записи)
	data := map[string]any{"walletId": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationType": "DEPOSIT", "amount": 500670}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/lala", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "incorrect ID value entered"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong3(t *testing.T) {
	// данные для запроса (некорректный walletId)
	data := map[string]any{"walletId": "hello_moscow", "operationType": "DEPOSIT", "amount": 500670}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong4(t *testing.T) {
	// данные для запроса (некорректный operationType)
	data := map[string]any{"walletId": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationType": "SELF", "amount": 500670}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(500), "message": "internal server error"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong5(t *testing.T) {
	// данные для запроса (отрицательный amount)
	data := map[string]any{"walletId": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationType": "DEPOSIT", "amount": -500670}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "invoice amount may be zero or greater"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong6(t *testing.T) {
	// данные для запроса (некорректный amount)
	data := map[string]any{"walletId": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationType": "DEPOSIT", "amount": "love"}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "invalid request"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong7(t *testing.T) {
	// данные для запроса (отсутствующий walletId)
	data := map[string]any{"operationType": "DEPOSIT", "amount": 30450}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "invalid request"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong8(t *testing.T) {
	// данные для запроса (отсутствующий operationType)
	data := map[string]any{"walletId": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "amount": 200350}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "invalid request"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestUpdateWalletWrong9(t *testing.T) {
	// данные для запроса (отсутствующий amount)
	data := map[string]any{"walletId": "c06a41c2-8063-4540-9558-6edf6ba9eafb", "operationType": "DEPOSIT"}
	reqData, _ := json.Marshal(data)
	// выполнение запроса
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/wallet/3", bytes.NewBuffer(reqData))
	// добавляем заголовок
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "invalid request"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestDeleteWalletRight(t *testing.T) {
	// выполнение запроса удаления
	reqDel, _ := http.NewRequest(http.MethodDelete, "/api/v1/wallet/6", nil)
	wDel := httptest.NewRecorder()
	router.ServeHTTP(wDel, reqDel)
	// проверка результатов
	assert.Equal(t, http.StatusOK, wDel.Code)
	want := "entry with ID 6 has been successfully deleted"
	assert.Equal(t, want, wDel.Body.String())
	// выполнение запроса получения оставшихся записей
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)
	// проверка результатов
	assert.Equal(t, http.StatusOK, wGet.Code)
	var responseGet []map[string]any
	err := json.Unmarshal(wGet.Body.Bytes(), &responseGet)
	assert.NoError(t, err)
	assert.Equal(t, 8, len(responseGet))
}

func TestDeleteWalletWrong1(t *testing.T) {
	// данные для запроса (несуществющий id записи)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/wallet/38", nil)
	// добавляем заголовок
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(404), "message": "no records with this ID were found"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}

func TestDeleteWalletWrong2(t *testing.T) {
	// данные для запроса (некорректный id записи)
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/wallet/c++", nil)
	// добавляем заголовок
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// проверка результатов
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	want := map[string]any{"code": float64(400), "message": "incorrect ID value entered"}
	assert.NoError(t, err)
	assert.Equal(t, want, response)
}
