-- +goose Up
CREATE TABLE IF NOT EXISTS wallet(
	id BIGSERIAL PRIMARY KEY,
	walletId UUID UNIQUE NOT NULL,
	operationType VARCHAR(10) NOT NULL,
	amount INT NOT NULL,
	CONSTRAINT chk_oper_type CHECK (operationType IN ('DEPOSIT', 'WITHDRAW'))
);

-- +goose Down
DROP TABLE IF EXISTS wallet;
