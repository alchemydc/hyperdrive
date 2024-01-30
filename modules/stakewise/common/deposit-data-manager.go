package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nodeset-org/eth-utils/beacon"
	"github.com/nodeset-org/hyperdrive/modules/common/validator/utils"
	swconfig "github.com/nodeset-org/hyperdrive/shared/config/modules/stakewise"
	"github.com/nodeset-org/hyperdrive/shared/types"
)

const (
	// Stakewise validators deposit a full 32 ETH
	StakewiseDepositAmount uint64      = 32e9
	fileMode               os.FileMode = 0600
)

// DepositDataManager manages the aggregated deposit data file that Stakewise uses
type DepositDataManager struct {
	dataPath string
	sp       *StakewiseServiceProvider
}

// Creates a new manager
func NewDepositDataManager(sp *StakewiseServiceProvider) *DepositDataManager {
	return &DepositDataManager{
		dataPath: filepath.Join(sp.GetModuleDir(), swconfig.DepositDataFile),
		sp:       sp,
	}
}

// Regenerates the deposit data file from all of the Stakewise validator keys in its keystore folder, and updates the deposit data file.
// Returns the total number of validator keys stored on disk.
func (m *DepositDataManager) RegenerateDepositData() (int, error) {
	resources := m.sp.GetResources()
	wallet := m.sp.GetWallet()

	// Stakewise uses the same withdrawal creds for each validator
	withdrawalCreds := utils.GetWithdrawalCredsFromAddress(resources.Vault)

	// Create the new aggregated deposit data for all generated keys
	keys, err := wallet.GetAllPrivateKeys()
	if err != nil {
		return 0, fmt.Errorf("error loading all validator keys: %w", err)
	}
	dataList := make([]*types.ExtendedDepositData, len(keys))
	for i, key := range keys {
		depositData, err := utils.GetDepositData(key, withdrawalCreds, resources.GenesisForkVersion, StakewiseDepositAmount, resources.Network)
		if err != nil {
			pubkey := beacon.ValidatorPubkey(key.PublicKey().Marshal())
			return 0, fmt.Errorf("error getting deposit data for key %s: %w", pubkey.Hex(), err)
		}
		dataList[i] = &depositData
	}

	// Serialize it
	bytes, err := json.Marshal(dataList)
	if err != nil {
		return 0, fmt.Errorf("error serializing deposit data: %w", err)
	}

	// Write it
	err = os.WriteFile(m.dataPath, bytes, fileMode)
	if err != nil {
		return 0, fmt.Errorf("error saving deposit data to disk: %w", err)
	}
	return len(dataList), nil
}
