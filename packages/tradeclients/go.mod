module github.com/crypto-exchange-lab/tradeclients

go 1.22

require github.com/crypto-exchange-lab/go-common v0.0.0

replace (
	github.com/crypto-exchange-lab/go-common => ../go-common
	github.com/crypto-exchange-lab/orderstore => ../orderstore
)
