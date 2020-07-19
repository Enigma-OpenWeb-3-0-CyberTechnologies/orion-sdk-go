package rest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.ibm.com/blockchaindb/library/pkg/crypto"
	"github.ibm.com/blockchaindb/protos/types"
	"github.ibm.com/blockchaindb/sdk/pkg/config"
	server "github.ibm.com/blockchaindb/sdk/pkg/database/mock"
)

func TestClient_GetState(t *testing.T) {
	server.StartTestServer()
	defer server.StopServer()
	opt := createOptions()
	userCrypto, err := opt.User.LoadCrypto(nil)
	rc, err := NewRESTClient("http://localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, rc)
	req := &types.GetStateQueryEnvelope{
		Payload: &types.GetStateQuery{
			UserID: "testUser",
			DBName: "testDb",
			Key:    "key1",
		},
		Signature: nil,
	}
	req.Signature, err = userCrypto.Sign(req.Payload)
	require.NoError(t, err)

	resp, err := rc.GetState(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Payload)
}

func TestClient_GetStateError(t *testing.T) {
	server.StartTestServer()
	defer server.StopServer()
	rc, err := NewRESTClient("http://localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, rc)
	req := &types.GetStateQueryEnvelope{
		Payload: &types.GetStateQuery{
			UserID: "testUser",
			DBName: "testDb",
			Key:    "key1",
		},
		Signature: nil,
	}

	resp, err := rc.GetState(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "empty signature")
}

func TestClient_GetStatus(t *testing.T) {
	server.StartTestServer()
	defer server.StopServer()
	opt := createOptions()
	userCrypto, err := opt.User.LoadCrypto(nil)
	rc, err := NewRESTClient("http://localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, rc)
	req := &types.GetStatusQueryEnvelope{
		Payload: &types.GetStatusQuery{
			UserID: "testUser",
			DBName: "testDb",
		},
		Signature: nil,
	}
	req.Signature, err = userCrypto.Sign(req.Payload)
	require.NoError(t, err)

	resp, err := rc.GetStatus(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Payload)
}

func TestClient_GetStatusError(t *testing.T) {
	server.StartTestServer()
	defer server.StopServer()
	rc, err := NewRESTClient("http://localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, rc)
	req := &types.GetStatusQueryEnvelope{
		Payload: &types.GetStatusQuery{
			UserID: "testUser",
			DBName: "testDb",
		},
		Signature: nil,
	}

	resp, err := rc.GetStatus(context.Background(), req)
	require.Error(t, err)
	require.Nil(t, resp)
	require.Contains(t, err.Error(), "empty signature")
}

func TestClient_SubmitTransaction(t *testing.T) {
	server.StartTestServer()
	defer server.StopServer()
	opt := createOptions()
	userCrypto, err := opt.User.LoadCrypto(nil)
	rc, err := NewRESTClient("http://localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, rc)

	req := &types.TransactionEnvelope{
		Payload: &types.Transaction{
			UserID:     []byte("testUser"),
			DBName:     "testDb",
			TxID:       []byte("TX1"),
			DataModel:  types.Transaction_KV,
			Statements: nil,
			Reads:      nil,
			Writes:     nil,
		},
		Signature: nil,
	}
	req.Signature, err = userCrypto.Sign(req.Payload)
	require.NoError(t, err)

	resp, err := rc.SubmitTransaction(context.Background(), req)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestNewRESTClient(t *testing.T) {
	rc, err := NewRESTClient("http://localhost:9999")
	require.NoError(t, err)
	require.NotNil(t, rc)
	require.EqualValues(t, "http://localhost:9999", rc.BaseURL.String())
}

func createOptions() *config.Options {
	connOpt := &config.ConnectionOption{
		URL: "http://localhost:9999/",
	}
	connOpts := make([]*config.ConnectionOption, 0)
	connOpts = append(connOpts, connOpt)
	userOpt := &crypto.IdentityOptions{
		UserID:       "testUser",
		CAFilePath:   "../database/testdata/ca_service.cert",
		CertFilePath: "../database/testdata/client.pem",
		KeyFilePath:  "../database/testdata/client.key",
	}
	return &config.Options{
		ConnectionOptions: connOpts,
		User:              userOpt,
		TxOptions: &config.TxOptions{
			TxIsolation:   config.Serializable,
			ReadOptions:   &config.ReadOptions{QuorumSize: 1},
			CommitOptions: &config.CommitOptions{QuorumSize: 1},
		},
	}
}
