module github.com/crypto-exchange-lab/perpservice

go 1.22

require (
	github.com/crypto-exchange-lab/go-common v0.0.0
	github.com/crypto-exchange-lab/perps v0.0.0
	github.com/crypto-exchange-lab/perpstore v0.0.0
	github.com/crypto-exchange-lab/tradeclients v0.0.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/shopspring/decimal v1.4.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace (
	github.com/crypto-exchange-lab/go-common => ../go-common
	github.com/crypto-exchange-lab/perps => ../perps
	github.com/crypto-exchange-lab/perpstore => ../perpstore
	github.com/crypto-exchange-lab/tradeclients => ../tradeclients
)
