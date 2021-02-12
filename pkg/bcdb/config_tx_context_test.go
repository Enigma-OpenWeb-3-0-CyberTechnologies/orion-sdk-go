package bcdb

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	sdkConfig "github.ibm.com/blockchaindb/sdk/pkg/config"
	"github.ibm.com/blockchaindb/server/pkg/server/testutils"
	"github.ibm.com/blockchaindb/server/pkg/types"
)

func TestConfigTxContext_GetClusterConfig(t *testing.T) {
	cryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "server"})
	testServer, err := setupTestServer(t, cryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	bcdb := createDBInstance(t, cryptoDir, serverPort)
	session := openUserSession(t, bcdb, "admin", cryptoDir)

	tx, err := session.ConfigTx()
	require.NoError(t, err)

	clusterConfig, err := tx.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)

	require.Equal(t, 1, len(clusterConfig.Nodes))
	require.Equal(t, "testNode1", clusterConfig.Nodes[0].ID)
	require.Equal(t, "127.0.0.1", clusterConfig.Nodes[0].Address)
	require.Equal(t, 0, int(clusterConfig.Nodes[0].Port))
	serverCertBytes, _ := testutils.LoadTestClientCrypto(t, cryptoDir, "server")
	require.Equal(t, serverCertBytes.Raw, clusterConfig.Nodes[0].Certificate)

	require.Equal(t, 1, len(clusterConfig.Admins))
	require.Equal(t, "admin", clusterConfig.Admins[0].ID)
	adminCertBytes, _ := testutils.LoadTestClientCrypto(t, cryptoDir, "admin")
	require.Equal(t, adminCertBytes.Raw, clusterConfig.Admins[0].Certificate)

	caCert, _ := testutils.LoadTestClientCA(t, cryptoDir, testutils.RootCAFileName)
	require.True(t, len(clusterConfig.CertAuthConfig.Roots) > 0)
	require.Equal(t, caCert.Raw, clusterConfig.CertAuthConfig.Roots[0])

	clusterConfig.Nodes = nil
	clusterConfig.Admins = nil
	clusterConfig.CertAuthConfig = nil
	clusterConfigAgain, err := tx.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfigAgain.Nodes, "it is a deep copy")
	require.NotNil(t, clusterConfigAgain.Admins, "it is a deep copy")
	require.NotNil(t, clusterConfigAgain.CertAuthConfig, "it is a deep copy")
}

func TestConfigTxContext_AddAdmin(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "admin2", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)

	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	adminCert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "admin")
	admin := &types.Admin{
		ID:          "admin",
		Certificate: adminCert.Raw,
	}

	admin2Cert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "admin2")
	admin2 := &types.Admin{ID: "admin2", Certificate: admin2Cert.Raw}

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	session1 := openUserSession(t, bcdb, "admin", clientCryptoDir)

	// Add admin2
	tx, err := session1.ConfigTx()
	require.NoError(t, err)
	require.NotNil(t, tx)

	err = tx.AddAdmin(admin)
	require.EqualError(t, err, "admin already exists in current config: admin")

	err = tx.AddAdmin(admin2)
	require.NoError(t, err)

	err = tx.AddAdmin(admin2)
	require.EqualError(t, err, "admin already exists in pending config: admin2")

	txID, receipt, err := tx.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	tx2, err := session1.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err := tx2.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Admins, 2)

	found, index := AdminExists("admin2", clusterConfig.Admins)
	require.True(t, found)

	require.EqualValues(t, clusterConfig.Admins[index].Certificate, admin2Cert.Raw)

	// do something with the new admin
	session2 := openUserSession(t, bcdb, "admin2", clientCryptoDir)
	tx3, err := session2.ConfigTx()
	require.NoError(t, err)
	clusterConfig2, err := tx3.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig2)
}

func TestConfigTxContext_DeleteAdmin(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "admin2", "admin3", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	adminCert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "admin")
	admin := &types.Admin{
		ID:          "admin",
		Certificate: adminCert.Raw,
	}

	admin2Cert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "admin2")
	admin3Cert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "admin3")

	admin2 := &types.Admin{ID: "admin2", Certificate: admin2Cert.Raw}
	admin3 := &types.Admin{ID: "admin3", Certificate: admin3Cert.Raw}

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	session1 := openUserSession(t, bcdb, "admin", clientCryptoDir)

	// Add admin2 & admin3
	tx1, err := session1.ConfigTx()
	require.NoError(t, err)
	require.NotNil(t, tx1)
	err = tx1.AddAdmin(admin2)
	require.NoError(t, err)
	err = tx1.AddAdmin(admin3)
	require.NoError(t, err)

	txID, receipt, err := tx1.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	tx, err := session1.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err := tx.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Admins, 3)

	// Remove an admin
	session2 := openUserSession(t, bcdb, "admin2", clientCryptoDir)
	tx2, err := session2.ConfigTx()
	require.NoError(t, err)
	err = tx2.DeleteAdmin(admin.ID)
	require.NoError(t, err)
	err = tx2.DeleteAdmin(admin.ID)
	require.EqualError(t, err, "admin does not exist in pending config: admin")
	err = tx2.DeleteAdmin("non-admin")
	require.EqualError(t, err, "admin does not exist in current config: non-admin")

	txID, receipt, err = tx2.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	// verify tx was successfully committed
	tx3, err := session2.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err = tx3.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Admins, 2)
	found, index := AdminExists("admin2", clusterConfig.Admins)
	require.True(t, found)
	require.EqualValues(t, clusterConfig.Admins[index].Certificate, admin2Cert.Raw)

	found, index = AdminExists("admin3", clusterConfig.Admins)
	require.True(t, found)
	require.EqualValues(t, clusterConfig.Admins[index].Certificate, admin3Cert.Raw)

	// session1 by removed admin cannot execute additional transactions
	tx4, err := session1.ConfigTx()
	require.EqualError(t, err, "failed to obtain server's certificate")
	require.Nil(t, tx4)
}

func TestConfigTxContext_UpdateAdmin(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "admin2", "adminUpdated", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	admin2Cert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "admin2")
	adminUpdatedCert, _ := testutils.LoadTestClientCrypto(t, clientCryptoDir, "adminUpdated")

	admin2 := &types.Admin{ID: "admin2", Certificate: admin2Cert.Raw}

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	session1 := openUserSession(t, bcdb, "admin", clientCryptoDir)

	// Add admin2
	tx1, err := session1.ConfigTx()
	require.NoError(t, err)
	require.NotNil(t, tx1)
	err = tx1.AddAdmin(admin2)
	require.NoError(t, err)

	txID, receipt, err := tx1.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	// Update an admin
	session2 := openUserSession(t, bcdb, "admin2", clientCryptoDir)
	tx2, err := session2.ConfigTx()
	require.NoError(t, err)
	err = tx2.UpdateAdmin(&types.Admin{ID: "admin", Certificate: adminUpdatedCert.Raw})
	require.NoError(t, err)
	err = tx2.UpdateAdmin(&types.Admin{ID: "non-admin", Certificate: []byte("bad-cert")})
	require.EqualError(t, err, "admin does not exist in current config: non-admin")

	txID, receipt, err = tx2.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	tx, err := session2.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err := tx.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Admins, 2)

	found, index := AdminExists("admin", clusterConfig.Admins)
	require.True(t, found)
	require.EqualValues(t, clusterConfig.Admins[index].Certificate, adminUpdatedCert.Raw)

	// session1 by updated admin cannot execute additional transactions, need to recreate session
	tx3, err := session1.ConfigTx()
	require.EqualError(t, err, "failed to obtain server's certificate")
	require.Nil(t, tx3)

	// need to recreate session with new credentials
	session3, err := bcdb.Session(&sdkConfig.SessionConfig{
		UserConfig: &sdkConfig.UserConfig{
			UserID:         "admin",
			CertPath:       path.Join(clientCryptoDir, "adminUpdated.pem"),
			PrivateKeyPath: path.Join(clientCryptoDir, "adminUpdated.key"),
		},
	})
	require.NoError(t, err)
	tx3, err = session3.ConfigTx()
	require.NoError(t, err)
	require.NotNil(t, tx3)
}

func TestConfigTxContext_AddClusterNode(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	session1 := openUserSession(t, bcdb, "admin", clientCryptoDir)
	tx, err := session1.ConfigTx()
	require.NoError(t, err)
	config, err := tx.GetClusterConfig()
	require.NoError(t, err)

	node2 := config.Nodes[0]
	node2.ID = "testNode2"
	node2.Port++
	err = tx.AddClusterNode(node2)
	require.NoError(t, err)

	txID, receipt, err := tx.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	tx2, err := session1.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err := tx2.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Nodes, 2)

	found, index := NodeExists("testNode2", clusterConfig.Nodes)
	require.True(t, found)
	require.Equal(t, clusterConfig.Nodes[index].Port, node2.Port)
}

func TestConfigTxContext_DeleteClusterNode(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	session1 := openUserSession(t, bcdb, "admin", clientCryptoDir)
	tx1, err := session1.ConfigTx()
	require.NoError(t, err)
	config, err := tx1.GetClusterConfig()
	require.NoError(t, err)

	id1 := config.Nodes[0].ID
	node2 := config.Nodes[0]
	node2.ID = "testNode2"
	node2.Port++

	err = tx1.AddClusterNode(node2)
	require.NoError(t, err)
	err = tx1.DeleteClusterNode(id1)
	require.NoError(t, err)

	txID, receipt, err := tx1.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	// verify tx was successfully committed. "Get" works once per Tx.
	tx, err := session1.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err := tx.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Nodes, 1)

	found, index := NodeExists("testNode2", clusterConfig.Nodes)
	require.True(t, found)
	require.Equal(t, clusterConfig.Nodes[index].Port, node2.Port)
}

func TestConfigTxContext_UpdateClusterNode(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	session1 := openUserSession(t, bcdb, "admin", clientCryptoDir)
	tx1, err := session1.ConfigTx()
	require.NoError(t, err)
	config, err := tx1.GetClusterConfig()
	require.NoError(t, err)

	node1 := config.Nodes[0]
	node1.Port++
	err = tx1.UpdateClusterNode(node1)
	require.NoError(t, err)

	txID, receipt, err := tx1.Commit(true)
	require.NoError(t, err)
	require.NotNil(t, txID)
	require.NotNil(t, receipt)

	// verify tx was successfully committed. "Get" works once per Tx.
	tx, err := session1.ConfigTx()
	require.NoError(t, err)
	clusterConfig, err := tx.GetClusterConfig()
	require.NoError(t, err)
	require.NotNil(t, clusterConfig)
	require.Len(t, clusterConfig.Nodes, 1)

	found, index := NodeExists("testNode1", clusterConfig.Nodes)
	require.True(t, found)
	require.Equal(t, clusterConfig.Nodes[index].Port, node1.Port)
}

func TestConfigTx_CommitAbortFinality(t *testing.T) {
	clientCryptoDir := testutils.GenerateTestClientCrypto(t, []string{"admin", "server"})
	testServer, err := setupTestServer(t, clientCryptoDir)
	defer func() {
		if testServer != nil {
			_ = testServer.Stop()
		}
	}()
	require.NoError(t, err)
	err = testServer.Start()
	require.NoError(t, err)

	serverPort, err := testServer.Port()
	require.NoError(t, err)

	bcdb := createDBInstance(t, clientCryptoDir, serverPort)
	for i := 0; i < 3; i++ {
		session := openUserSession(t, bcdb, "admin", clientCryptoDir)
		tx, err := session.ConfigTx()
		require.NoError(t, err)

		config, err := tx.GetClusterConfig()
		require.NoError(t, err)
		node1 := config.Nodes[0]
		node1.Port++
		nodeId := node1.ID
		nodePort := node1.Port
		err = tx.UpdateClusterNode(config.Nodes[0])
		require.NoError(t, err)

		assertTxFinality(t, TxFinality(i), tx, session)

		config, err = tx.GetClusterConfig()
		require.EqualError(t, err, ErrTxSpent.Error())
		require.Nil(t, config)

		err = tx.AddClusterNode(&types.NodeConfig{})
		require.EqualError(t, err, ErrTxSpent.Error())
		err = tx.DeleteClusterNode("id")
		require.EqualError(t, err, ErrTxSpent.Error())
		err = tx.UpdateClusterNode(&types.NodeConfig{})
		require.EqualError(t, err, ErrTxSpent.Error())

		err = tx.AddAdmin(&types.Admin{})
		require.EqualError(t, err, ErrTxSpent.Error())
		err = tx.DeleteAdmin("id")
		require.EqualError(t, err, ErrTxSpent.Error())
		err = tx.UpdateAdmin(&types.Admin{})
		require.EqualError(t, err, ErrTxSpent.Error())

		if TxFinality(i) != TxFinalityAbort {
			tx, err = session.ConfigTx()
			require.NoError(t, err)

			config, err := tx.GetClusterConfig()
			require.NoError(t, err)
			node1 := config.Nodes[0]
			require.Equal(t, nodeId, node1.ID)
			require.Equal(t, nodePort, node1.Port)
		}
	}
}
