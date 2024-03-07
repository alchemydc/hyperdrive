package wallet

import (
	"errors"
	"fmt"
	"net/url"
	_ "time/tzdata"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/gorilla/mux"
	"github.com/nodeset-org/hyperdrive/shared/types/api"
	"github.com/nodeset-org/hyperdrive/shared/utils/input"
	nmc_server "github.com/rocket-pool/node-manager-core/api/server"
)

// ===============
// === Factory ===
// ===============

type walletSignMessageContextFactory struct {
	handler *WalletHandler
}

func (f *walletSignMessageContextFactory) Create(args url.Values) (*walletSignMessageContext, error) {
	c := &walletSignMessageContext{
		handler: f.handler,
	}
	inputErrs := []error{
		nmc_server.ValidateArg("message", args, input.ValidateByteArray, &c.message),
	}
	return c, errors.Join(inputErrs...)
}

func (f *walletSignMessageContextFactory) RegisterRoute(router *mux.Router) {
	nmc_server.RegisterQuerylessGet[*walletSignMessageContext, api.WalletSignMessageData](
		router, "sign-message", f, f.handler.serviceProvider.ServiceProvider,
	)
}

// ===============
// === Context ===
// ===============

type walletSignMessageContext struct {
	handler *WalletHandler
	message []byte
}

func (c *walletSignMessageContext) PrepareData(data *api.WalletSignMessageData, opts *bind.TransactOpts) error {
	sp := c.handler.serviceProvider
	w := sp.GetWallet()

	err := errors.Join(
		sp.RequireWalletReady(),
	)
	if err != nil {
		return err
	}

	signedBytes, err := w.SignMessage(c.message)
	if err != nil {
		return fmt.Errorf("error signing message: %w", err)
	}
	data.SignedMessage = signedBytes
	return nil
}
