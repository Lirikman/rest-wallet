# WALLET


## Description
Rest-service for working with a virtual wallet

Example of 'Wallet' object:
* **id (bigserial, read-only):** Unique record number
* **walletId (uuid, required):** Unique wallet identifier
* **operationType (string, required):** Wallet transaction type, supports DEPOSIT or WITHDRAW
* **amount (int):** Amount of money in the wallet


## 🛠️ Tech Stack

### Core Stack
* **Language:** **Go (Golang)**
* **Web Framework:** **Gin Gonic**
* **Database:** **PostgreSQL**

### Database & Development Tools
* **Data Access Layer:** **sqlc**
* **Database Migrations:** **goose**

### API & Environment
* **Containerization:** **Docker Compose** 

## 💻 Prerequisites
To start the project you will need:
 * Docker Engine v24.0+
 * Docker Compose v2.20+

## 🚀 Quick Start
Follow these steps to run the application locally:

1. Clone the repository
```bash
git clone https://github.com/Lirikman/rest-wallet.git
cd rest-wallet
```

2. Launch the project
```bash
make run
```

3. Check the work

  After a successful launch, the application will be available at: 
👉 http://127.0.0.1:8080/api/v1

5. Stopping and removing the container
```bash
make stop
```


## 🛠️ Development

### 🧪 Run golangci-lint 

```bash
make lint
```

### 🧪 Run tests

```bash
make test
```

## 📡 API documentation

All requests are sent to the base URL: http://127.0.0.1:8080/api/v1

Headers: Content-Type: application/json


### Getting all wallet records
Returns all wallet records.

**GET** /wallets
```json
[
    {
    "id": 1,
    "walletid": "81997f52-03eb-42ac-89d8-e55d26b09003",
    "operationtype": "DEPOSIT",
    "amount": 350000
  },
  {
    "id": 2,
    "walletid": "92758591-6720-44de-a97a-3bb1d00a961a",
    "operationtype": "WITHDRAW",
    "amount": 453600
  },
  {
    "id": 3,
    "walletid": "d7ad1bf4-244f-43ad-934b-e21e83fcf2c8",
    "operationtype": "WITHDRAW",
    "amount": 1020040
  }
]
```
Response code: 200 OK

### Getting a list of wallets in the selected range
Returns a list of wallets in the selected range.

**GET** /wallets?range=[2,5]
```json
[
  {
    "id": 3,
    "walletid": "d7ad1bf4-244f-43ad-934b-e21e83fcf2c8",
    "operationtype": "WITHDRAW",
    "amount": 1020040
  },
  {
    "id": 4,
    "walletid": "8883e024-8c4c-470b-9af0-f34dc12f69d8",
    "operationtype": "WITHDRAW",
    "amount": 2334510
  },
  {
    "id": 5,
    "walletid": "61eca831-525e-4b9e-b859-92d66e307405",
    "operationtype": "WITHDRAW",
    "amount": 444444
  }
]
```
Response code: 200 OK

### Creating a new wallet entry
Creates a new virtual wallet entry

Requirements:

- All records are unique.
- The 'walletId' field must be in the UUID format, not empty, unique.
- The 'operationType' field - string, not empty. Supports values ​​- DEPOSIT and WITHDRAW.
- The 'amount' field is a positive integer, optional. Default is 0.

**POST** /wallets

**Request body example:**
```json
{
  "walletId":"81997f52-03eb-42ac-89d8-e55d26b09003",
  "operationType":"DEPOSIT", 
  "amount":350000
}
```

or

```json
{
  "walletId":"d7ad1bf4-244f-43ad-934b-e21e83fcf2c8", 
  "operationType":"WITHDRAW",
  "amount":1020040
}
```

or

```json
{
  "walletId":"61eca831-525e-4b9e-b859-92d66e30750a",
  "operationType":"DEPOSIT"
}
```

**Example answer:**
```json
{
  "id": 5,
  "walletid":"d7ad1bf4-244f-43ad-934b-e21e83fcf2c8", 
  "operationtype":"WITHDRAW",
  "amount":1020040
}
```
Response code: 201 Created

or

```json
{
  "id": 6,
  "walletid":"61eca831-525e-4b9e-b859-92d66e30750a",
  "operationtype":"DEPOSIT"
  "amount":0
}
```
Response code: 201 Created

### Getting a wallet record by ID
Returns a wallet entry or an error about the absence of a wallet.

**GET** /wallets/by-id/2

**Example answer:**
```json
{
  "id": 2,
  "walletid": "92758591-6720-44de-a97a-3bb1d00a961a",
  "operationtype": "WITHDRAW",
  "amount": 453600
}
```
Response code: 200 OK

or

```json
{
  "code": 404,
  "message": "no records with this ID were found"
}
```
Response code: 404 NotFound

### Getting your wallet balance by walletId
Returns a message about the wallet balance or an error if there is no entry.

**GET** /wallets/81997f52-03eb-42ac-89d8-e55d26b09003

**Example answer:**
```
  "balance of the wallet with the WaletId 92758591-6720-44de-a97a-3bb1d00a961a is 453600 rub."
```
Response code: 200 OK

or

```json
{
  "code":400,
  "message":"wallet id is incorrect (example wallet_id: 123e4567-e89b-12d3-a456-426655440000)"
}
```
Response code: 400 Bad Request

### Updating wallet information
Updating wallet information.

**PUT** /wallets/3

**Request body example:**
```json
{
  "walletId":"81b84a74-5735-4f01-8e77-f2bd0e137bec",
  "operationType":"WITHDRAW",
  "amount":100200
}
```

or

```json
{
  "be804750-4fa3-47f9-ba62-d991afea3645",
  "operationType":"DEPOSIT"
}
```

**Example answer:**
```json
{
  "walletid":"81b84a74-5735-4f01-8e77-f2bd0e137bec",
  "operationtype":"WITHDRAW",
  "amount":100200
}
```
Response code: 200 OK

### Deleting a wallet entry
Deletes a wallet entry by its ID.

**DELETE** /wallets/5

**Example answer:**
```
  "entry with ID 5 has been successfully deleted"
```
Response code: 200 OK
