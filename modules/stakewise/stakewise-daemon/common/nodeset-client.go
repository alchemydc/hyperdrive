package swcommon

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/goccy/go-json"

	"github.com/nodeset-org/eth-utils/beacon"
	"github.com/nodeset-org/eth-utils/common"
	swshared "github.com/nodeset-org/hyperdrive/modules/stakewise/shared"
	"github.com/nodeset-org/hyperdrive/shared/types"
)

const (
	// The message to sign with the node wallet when uploading deposit data
	nodesetAuthMessage string = "nodesetdev"

	// Header used for the wallet signature during a deposit data upload
	authHeader string = "Authorization"

	// API paths
	depositDataPath string = "deposit-data"
	metaPath        string = "meta"
	validatorsPath  string = "validators"
)

// =================
// === Responses ===
// =================

// api/deposit-data/meta
type DepositDataMetaResponse struct {
	Version uint64 `json:"version"`
}

// api/deposit-data
type DepositDataResponse struct {
	Version uint64                      `json:"version"`
	Data    []types.ExtendedDepositData `json:"data"`
}

// api/validators
type ValidatorsResponse struct {
	Data []beacon.ValidatorPubkey `json:"data"`
}

// ==============
// === Client ===
// ==============

// Client for interacting with the Nodeset server
type NodesetClient struct {
	sp            *StakewiseServiceProvider
	res           *swshared.StakewiseResources
	debug         bool
	authSignature []byte
}

// Creates a new Nodeset client
func NewNodesetClient(sp *StakewiseServiceProvider) *NodesetClient {
	cfg := sp.GetConfig()
	return &NodesetClient{
		sp:    sp,
		res:   sp.GetResources(),
		debug: cfg.DebugMode.Value,
	}
}

// Checks if the NodeSet authorization signature has been set, and if not, creates it by getting a signed message from the node wallet
func (c *NodesetClient) EnsureAuthSignatureExists() error {
	if c.authSignature != nil {
		return nil
	}

	// Sign the auth message
	hd := c.sp.GetHyperdriveClient()
	signResponse, err := hd.Wallet.SignMessage([]byte(nodesetAuthMessage))
	if err != nil {
		return fmt.Errorf("error signing authorization message: %w", err)
	}
	c.authSignature = signResponse.Data.SignedMessage
	return nil
}

// Uploads deposit data to Nodeset
func (c *NodesetClient) UploadDepositData(depositData []byte) ([]byte, error) {
	response, err := c.submitRequest(http.MethodPost, bytes.NewBuffer(depositData), nil, depositDataPath)
	if err != nil {
		return nil, fmt.Errorf("error uploading deposit data: %w", err)
	}
	return response, nil
}

// Get the current version of the aggregated deposit data on the server
func (c *NodesetClient) GetServerDepositDataVersion() (uint64, error) {
	response, err := c.submitRequest(http.MethodGet, nil, nil, depositDataPath, metaPath)
	if err != nil {
		return 0, fmt.Errorf("error getting deposit data version: %w", err)
	}

	var body DepositDataMetaResponse
	err = json.Unmarshal(response, &body)
	if err != nil {
		return 0, fmt.Errorf("error deserializing deposit data version response: %w", err)
	}
	return body.Version, nil
}

// Get the aggregated deposit data from the server
func (c *NodesetClient) GetServerDepositData() (uint64, []types.ExtendedDepositData, error) {
	params := map[string]string{
		"vault": c.res.Vault.Hex(),
	}
	response, err := c.submitRequest(http.MethodGet, nil, params, depositDataPath)
	if err != nil {
		return 0, nil, fmt.Errorf("error getting deposit data: %w", err)
	}

	var body DepositDataResponse
	err = json.Unmarshal(response, &body)
	if err != nil {
		return 0, nil, fmt.Errorf("error deserializing deposit data response: %w", err)
	}
	return body.Version, body.Data, nil
}

// Get a list of all of the pubkeys that have already been registered with NodeSet for this node
func (c *NodesetClient) GetRegisteredValidators() ([]beacon.ValidatorPubkey, error) {
	response, err := c.submitRequest(http.MethodGet, nil, nil, validatorsPath)
	if err != nil {
		return nil, fmt.Errorf("error getting registered validators: %w", err)
	}

	var body ValidatorsResponse
	err = json.Unmarshal(response, &body)
	if err != nil {
		return nil, fmt.Errorf("error deserializing registered validators response: %w", err)
	}
	return body.Data, nil
}

// Send a request to the server and read the response
func (c *NodesetClient) submitRequest(method string, body io.Reader, queryParams map[string]string, subroutes ...string) ([]byte, error) {
	// Make the request
	path, err := url.JoinPath(c.res.NodesetApiUrl, subroutes...)
	if err != nil {
		return nil, fmt.Errorf("error joining path [%v]: %w", subroutes, err)
	}
	request, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, fmt.Errorf("error generating request to [%s]: %w", path, err)
	}
	for name, value := range queryParams {
		request.URL.Query().Add(name, value)
	}

	// Set the headers
	err = c.EnsureAuthSignatureExists()
	if err != nil {
		return nil, fmt.Errorf("initializing authorization signature failed: %w", err)
	}
	request.Header.Set(authHeader, common.EncodeHexWithPrefix(c.authSignature))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	// Upload it to the server
	if c.debug {
		fmt.Printf("Sending NodeSet server request => %s\n", path)
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error submitting request to nodeset server: %w", err)
	}

	// Read the body
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)

	// Check if the request failed
	if resp.StatusCode != http.StatusOK {
		if err != nil {
			return nil, fmt.Errorf("nodeset server responded to request with code %s but reading the response body failed: %w", resp.Status, err)
		}
		msg := string(bytes)
		return nil, fmt.Errorf("nodeset server responded to request with code %s: [%s]", resp.Status, msg)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading the response body for nodeset request: %w", err)
	}

	// Debug log
	if c.debug {
		fmt.Printf("NodeSet response <= %s\n", string(bytes))
	}
	return bytes, nil
}