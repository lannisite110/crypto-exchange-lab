package chainrpc

// Block is a subset of eth_getBlockByNumber JSON.
type Block struct {
	Number       string        `json:"number"`
	Hash         string        `json:"hash"`
	ParentHash   string        `json:"parentHash"`
	Timestamp    string        `json:"timestamp"`
	Transactions []interface{} `json:"transactions"`
}

// Transaction is eth_getTransactionByHash JSON.
type Transaction struct {
	Hash        string `json:"hash"`
	BlockNumber string `json:"blockNumber"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	TransactionIndex string `json:"transactionIndex"`
}

// Receipt is eth_getTransactionReceipt JSON.
type Receipt struct {
	TransactionHash string `json:"transactionHash"`
	Status          string `json:"status"`
	GasUsed         string `json:"gasUsed"`
	Logs            []Log  `json:"logs"`
}

// Log is an EVM log entry.
type Log struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber string   `json:"blockNumber"`
	TxHash      string   `json:"transactionHash"`
	LogIndex    string   `json:"logIndex"`
}

// LogFilter for eth_getLogs.
type LogFilter struct {
	FromBlock string   `json:"fromBlock"`
	ToBlock   string   `json:"toBlock"`
	Address   []string `json:"address,omitempty"`
	Topics    [][]string `json:"topics,omitempty"`
}
