-- name: ListWallets :many
SELECT * FROM wallet
ORDER BY id
LIMIT $1 OFFSET $2;

-- name: GetBalanceFromWalletUUID :one
SELECT amount 
FROM wallet
WHERE walletId = $1;

-- name: GetWalletFromId :one
SELECT id, walletId, operationType, amount
FROM wallet
WHERE id = $1;

-- name: CreateWallet :one
INSERT INTO wallet(
walletId, operationType, amount
) VALUES (
$1, $2, $3
)
RETURNING id, walletId, operationType, amount;

-- name: UpdateWalletNoBalance :one
UPDATE wallet
SET walletId = $2, operationType = $3
WHERE id = $1
RETURNING id, walletId, operationType, amount;

-- name: UpdateWalletBalance :one
UPDATE wallet
SET walletId = $2, operationType = $3, amount = $4
WHERE id = $1
RETURNING id, walletId, operationType, amount;

-- name: DeleteWallet :exec
DELETE FROM wallet
WHERE id = $1;