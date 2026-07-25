package wallet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	generated "github.com/Lirikman/rest-wallet/db/generated"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

// DTO для запроса создания записи
type CreateWalletReqDTO struct {
	WalletID      string `json:"walletId" binding:"required"`
	OperationType string `json:"operationType" binding:"required"`
	Amount        int32  `json:"amount"`
}

// DTO для обновления записи
type UpdateWalletReqDTO struct {
	WalletID      string `json:"walletId" binding:"required"`
	Operationtype string `json:"operationType" binding:"required"`
	Amount        *int32 `json:"amount"`
}

// структура ошибки при запросах
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// обработчик создания новой записи
func CreateWallet(db *generated.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateWalletReqDTO

		// десериализация данных тела запроса
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
			return
		}

		// проверяем wallet_id на корректность
		var walletUUID pgtype.UUID
		err := walletUUID.Scan(req.WalletID)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"})
			return
		}

		// проверяем корректность ввода баланса
		if req.Amount < 0 {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "amount can range from zero and above"})
			return
		}

		// формируем параметры для запроса
		params := generated.CreateWalletParams{
			Walletid:      walletUUID,
			Operationtype: req.OperationType,
			Amount:        &req.Amount,
		}

		// создаём контекст с таймаутом 2 сек.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		// освобождаем ресурсы контекста
		defer cancel()

		// создаём новую запись в БД
		wall, err := db.CreateWallet(ctx, params)
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - create new wallet %s: %v\n", req.WalletID, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error create new wallet: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		c.JSON(http.StatusCreated, wall)
	}
}

// обработчик получения всех записей
func ListWallets(db *generated.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var paginParams generated.ListWalletsParams
		// получаем параметры для пагинации
		rangeLinks := c.DefaultQuery("range", "[0,50]")
		// задаём параметры по умолчанию
		limit := 50
		offset := 0
		// задаём регулярное выражение для поиска всех чисел
		re := regexp.MustCompile(`-?\d+`)
		// получаем лимит записей на странице и сдвиг для вывода записей
		numRange := re.FindAllString(rangeLinks, -1)
		// проверяем корректность ввода данных
		if len(numRange) != 2 {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "range must be specified by two numbers, example: [1,4]"})
			return
		}
		idx0, _ := strconv.Atoi(numRange[0])
		idx1, _ := strconv.Atoi(numRange[1])
		// проверка на положительные значения
		if idx0 < 0 || idx1 < 0 {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "the range value must be positive"})
			return
		}
		// если первый индекс меньше второго
		if idx0 < idx1 {
			limit = idx1 - idx0
			offset = idx0
		}
		// если индексы равны
		if idx0 == idx1 {
			limit = 1
			offset = idx0
		}
		// если первый индекс больше второго
		if idx0 > idx1 {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "range values are specified incorrectly"})
			return
		}
		paginParams.Limit = int32(limit)
		paginParams.Offset = int32(offset)

		// создаём контекст с таймаутом 2 сек.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		// освобождаем ресурсы контекста
		defer cancel()

		// получаем все записи из БД с учётом пагинации
		wallets, err := db.ListWallets(ctx, paginParams)
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - getting list records: %v\n", err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error getting list of wallets: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		// проверяем, вызвана ли ошибка отсутствием записей в БД
		if len(wallets) == 0 {
			c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "no wallets found"})
			return
		}
		c.JSON(http.StatusOK, wallets)
	}
}

// обработчик получения записи по id
func GetWalletFromId(db *generated.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		// чтение параметра id из запроса
		idStr := c.Param("id")
		// проверяем id на корректность
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "incorrect ID value entered"})
			return
		}

		// создаём контекст с таймаутом 2 сек.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		// освобождаем ресурсы контекста
		defer cancel()

		// получаем запись по id
		wall, err := db.GetWalletFromId(ctx, id)
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - getting wallet by id %d: %v\n", id, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// проверяем, вызвана ли ошибка отсутствием строки в БД
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "no records with this ID were found"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error retrieving a record from the database by ID: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		c.JSON(http.StatusOK, wall)
	}
}

// обработчик получения записи по walletId
func GetBalanceFromWalletId(db *generated.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var walletUUID pgtype.UUID
		// чтение параметра user_id из запроса
		walletIdStr := c.Param("wallet_uuid")

		//проверяем wallet_id на корректность
		err := walletUUID.Scan(walletIdStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"})
			return
		}

		// создаём контекст с таймаутом 2 сек.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		// освобождаем ресурсы контекста
		defer cancel()

		// получааем запись по wallet_id
		bal, err := db.GetBalanceFromWalletUUID(ctx, walletUUID)
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - getting balance by wallet_id %s: %v\n", walletUUID, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// проверяем, вызвана ли ошибка отсутствием строки в БД
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "wallet with this wallet_id not found"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error retrieving a record from the database by wallet_id: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		message := fmt.Sprintf("balance of the wallet with the WaletId %v is %v rub.", walletUUID, bal)
		c.String(http.StatusOK, message)
	}
}

// обработчик обновления записи по id
func UpdateWallet(db *generated.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateWalletReqDTO
		var walletUUID pgtype.UUID
		// чтение параметра id из запроса
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "incorrect ID value entered"})
			return
		}

		// создаём контекст с таймаутом 2 сек.
		ctxGet, cancelGet := context.WithTimeout(c.Request.Context(), 2*time.Second)

		// проверяем наличие записи в БД
		_, err = db.GetWalletFromId(ctxGet, id)
		// освобождаем ресурсы контекста
		cancelGet()
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - getting wallet by id %d: %v\n", id, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// проверяем, вызвана ли ошибка отсутствием строки в БД
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "no records with this ID were found"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error retrieving a record from the database by ID: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		// десериализация данных тела запроса
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invalid request"})
			return
		}
		// проверяем wallet_id на корректность
		err = walletUUID.Scan(req.WalletID)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"})
			return
		}
		// проверяем поле amount на null
		if req.Amount == nil {
			// формируем параметры для запроса
			params := generated.UpdateWalletNoBalanceParams{
				ID:            id,
				Walletid:      walletUUID,
				Operationtype: req.Operationtype,
			}
			// создаём контекст с таймаутом 2 сек.
			ctxUpd, cancelUpd := context.WithTimeout(c.Request.Context(), 2*time.Second)
			// освобождаем ресурсы контекста
			defer cancelUpd()
			// обновляем запись, не трогая amount
			res, err := db.UpdateWalletNoBalance(ctxUpd, params)
			if err != nil {
				// проверяем, произошла ли ошибка из-за превышения таймаута
				if errors.Is(err, context.DeadlineExceeded) {
					log.Printf("database timeout - update wallet by id %d: %v\n", id, err)
					c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
					return
				}
				// иначе это другая ошибка сервера
				log.Printf("wallet update error: %v\n", err)
				c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
				return
			}
			c.JSON(http.StatusOK, res)
		}

		// проверяем корректность ввода баланса
		if *req.Amount < 0 {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "invoice amount may be zero or greater"})
			return
		}
		// формируем параметры для запроса
		params := generated.UpdateWalletBalanceParams{
			ID:            id,
			Walletid:      walletUUID,
			Operationtype: req.Operationtype,
			Amount:        req.Amount,
		}

		// создаём контекст с таймаутом 2 сек.
		ctxUpd, cancelUpd := context.WithTimeout(c.Request.Context(), 2*time.Second)
		// освобождаем ресурсы контекста
		defer cancelUpd()

		// обновляем запись
		res, err := db.UpdateWalletBalance(ctxUpd, params)
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - update wallet by id %d: %v\n", id, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("wallet update error: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

// обработчик удаления записи по id
func DeleteWallet(db *generated.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		// чтение параметра id из запроса
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, HTTPError{Code: http.StatusBadRequest, Message: "incorrect ID value entered"})
			return
		}

		// создаём контекст с таймаутом 2 сек.
		ctxGet, cancelGet := context.WithTimeout(c.Request.Context(), 2*time.Second)

		// проверяем наличие записи в БД
		_, err = db.GetWalletFromId(ctxGet, id)
		// освобождаем ресурсы контекста
		cancelGet()
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - get wallet by id %d: %v\n", id, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// проверяем, вызвана ли ошибка отсутствием строки в БД
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, HTTPError{Code: http.StatusNotFound, Message: "no records with this ID were found"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error retrieving a record from the database by ID: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}

		// создаём контекст с таймаутом 2 сек.
		ctxDel, cancelDel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		// освобождаем ресурсы контекста
		defer cancelDel()

		// удаляем запись
		err = db.DeleteWallet(ctxDel, id)
		if err != nil {
			// проверяем, произошла ли ошибка из-за превышения таймаута
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("database timeout - delete wallet by id %d: %v\n", id, err)
				c.JSON(http.StatusGatewayTimeout, HTTPError{Code: http.StatusGatewayTimeout, Message: "database timeout"})
				return
			}
			// иначе это другая ошибка сервера
			log.Printf("error deleting wallet subscription: %v\n", err)
			c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "internal server error"})
			return
		}
		message := fmt.Sprintf("entry with ID %s has been successfully deleted", idStr)
		c.String(http.StatusOK, message)
	}
}
