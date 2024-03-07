package client

import (
	"github.com/fatih/color"
	nmc_client "github.com/rocket-pool/node-manager-core/api/client"
)

const (
	jsonContentType string          = "application/json"
	apiColor        color.Attribute = color.FgHiCyan
)

// Binder for the Hyperdrive daemon API server
type ApiClient struct {
	context *nmc_client.RequesterContext
	Service *ServiceRequester
	Tx      *TxRequester
	Utils   *UtilsRequester
	Wallet  *WalletRequester
}

// Creates a new API client instance
func NewApiClient(baseRoute string, socketPath string, debugMode bool) *ApiClient {
	context := nmc_client.NewRequesterContext(baseRoute, socketPath, debugMode)

	client := &ApiClient{
		context: context,
		Service: NewServiceRequester(context),
		Tx:      NewTxRequester(context),
		Utils:   NewUtilsRequester(context),
		Wallet:  NewWalletRequester(context),
	}

	return client
}

// Set debug mode
func (c *ApiClient) SetDebug(debug bool) {
	c.context.DebugMode = debug
}
