package wallet

import (
	"net/url"
	_ "time/tzdata"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"
	"github.com/rocket-pool/node-manager-core/api/server"
	nmc_types "github.com/rocket-pool/node-manager-core/api/types"
)

// ===============
// === Factory ===
// ===============

type walletRestoreAddressContextFactory struct {
	handler *WalletHandler
}

func (f *walletRestoreAddressContextFactory) Create(args url.Values) (*walletRestoreAddressContext, error) {
	c := &walletRestoreAddressContext{
		handler: f.handler,
	}
	return c, nil
}

func (f *walletRestoreAddressContextFactory) RegisterRoute(router *mux.Router) {
	server.RegisterQuerylessGet[*walletRestoreAddressContext, nmc_types.SuccessData](
		router, "restore-address", f, f.handler.serviceProvider.ServiceProvider,
	)
}

// ===============
// === Context ===
// ===============

type walletRestoreAddressContext struct {
	handler *WalletHandler
}

func (c *walletRestoreAddressContext) PrepareData(data *nmc_types.SuccessData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	w := sp.GetWallet()

	return w.RestoreAddressToWallet()
}
