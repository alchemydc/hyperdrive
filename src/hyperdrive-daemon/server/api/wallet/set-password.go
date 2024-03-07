package wallet

import (
	"errors"
	"net/url"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"
	"github.com/nodeset-org/hyperdrive/daemon-utils/server"
	"github.com/nodeset-org/hyperdrive/shared/types/api"
	"github.com/nodeset-org/hyperdrive/shared/utils/input"
	nmc_server "github.com/rocket-pool/node-manager-core/api/server"
)

// ===============
// === Factory ===
// ===============

type walletSetPasswordContextFactory struct {
	handler *WalletHandler
}

func (f *walletSetPasswordContextFactory) Create(args url.Values) (*walletSetPasswordContext, error) {
	c := &walletSetPasswordContext{
		handler: f.handler,
	}
	inputErrs := []error{
		server.ValidateArg("password", args, input.ValidateNodePassword, &c.password),
		server.ValidateArg("save", args, input.ValidateBool, &c.save),
	}
	return c, errors.Join(inputErrs...)
}

func (f *walletSetPasswordContextFactory) RegisterRoute(router *mux.Router) {
	nmc_server.RegisterQuerylessGet[*walletSetPasswordContext, api.SuccessData](
		router, "set-password", f, f.handler.serviceProvider.ServiceProvider,
	)
}

// ===============
// === Context ===
// ===============

type walletSetPasswordContext struct {
	handler  *WalletHandler
	password string
	save     bool
}

func (c *walletSetPasswordContext) PrepareData(data *api.SuccessData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	w := sp.GetWallet()

	return w.SetPassword(c.password, c.save)
}
