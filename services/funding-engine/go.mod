module github.com/crypto-exchange-lab/funding-engine

go 1.22

require (
	github.com/crypto-exchange-lab/go-common v0.0.0
	github.com/crypto-exchange-lab/perpservice v0.0.0
	github.com/crypto-exchange-lab/perpstore v0.0.0
	github.com/crypto-exchange-lab/tradeclients v0.0.0
	github.com/shopspring/decimal v1.4.0
	go.uber.org/zap v1.27.0
)

replace (
	github.com/crypto-exchange-lab/go-common => ../../packages/go-common
	github.com/crypto-exchange-lab/perps => ../../packages/perps
	github.com/crypto-exchange-lab/perpservice => ../../packages/perpservice
	github.com/crypto-exchange-lab/perpstore => ../../packages/perpstore
	github.com/crypto-exchange-lab/tradeclients => ../../packages/tradeclients
)
