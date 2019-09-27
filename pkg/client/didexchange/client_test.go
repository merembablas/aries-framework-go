/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package didexchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/dispatcher"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/internal/mock/common/did"
	mockdispatcher "github.com/hyperledger/aries-framework-go/pkg/internal/mock/didcomm/dispatcher"
	mockprotocol "github.com/hyperledger/aries-framework-go/pkg/internal/mock/didcomm/protocol"
	mockprovider "github.com/hyperledger/aries-framework-go/pkg/internal/mock/provider"
	mockstore "github.com/hyperledger/aries-framework-go/pkg/internal/mock/storage"
	mockwallet "github.com/hyperledger/aries-framework-go/pkg/internal/mock/wallet"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
)

func TestNew(t *testing.T) {
	t.Run("test new client", func(t *testing.T) {
		_, err := New(&mockprovider.Provider{ServiceValue: didexchange.New(nil, &did.MockDIDCreator{}, &mockProvider{})})
		require.NoError(t, err)
	})

	t.Run("test error from get service from context", func(t *testing.T) {
		_, err := New(&mockprovider.Provider{ServiceErr: fmt.Errorf("service error")})
		require.Error(t, err)
		require.Contains(t, err.Error(), "service error")
	})

	t.Run("test error from cast service", func(t *testing.T) {
		_, err := New(&mockprovider.Provider{ServiceValue: nil})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cast service to DIDExchange Service failed")
	})
}

func TestClient_CreateInvitation(t *testing.T) {
	t.Run("test success", func(t *testing.T) {
		c, err := New(&mockprovider.Provider{ServiceValue: didexchange.New(nil, &did.MockDIDCreator{}, &mockProvider{}),
			WalletValue: &mockwallet.CloseableWallet{}, InboundEndpointValue: "endpoint"})
		require.NoError(t, err)
		inviteReq, err := c.CreateInvitation("agent")
		require.NoError(t, err)
		require.NotNil(t, inviteReq)
		require.NotEmpty(t, inviteReq.Label)
		require.NotEmpty(t, inviteReq.ID)
		require.Equal(t, "endpoint", inviteReq.ServiceEndpoint)
	})

	t.Run("test error from createSigningKey", func(t *testing.T) {
		c, err := New(&mockprovider.Provider{ServiceValue: didexchange.New(nil, &did.MockDIDCreator{}, &mockProvider{}),
			WalletValue: &mockwallet.CloseableWallet{CreateSigningKeyErr: fmt.Errorf("createSigningKeyErr")}})
		require.NoError(t, err)
		_, err = c.CreateInvitation("agent")
		require.Error(t, err)
		require.Contains(t, err.Error(), "createSigningKeyErr")
	})
}

func TestClient_QueryConnectionByID(t *testing.T) {
	c, err := New(&mockprovider.Provider{ServiceValue: didexchange.New(nil, &did.MockDIDCreator{}, &mockProvider{})})
	require.NoError(t, err)

	result, err := c.QueryConnectionByID("sample-id")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.ConnectionID)
}

func TestClient_RemoveConnection(t *testing.T) {
	c, err := New(&mockprovider.Provider{ServiceValue: didexchange.New(nil, &did.MockDIDCreator{}, &mockProvider{})})
	require.NoError(t, err)

	err = c.RemoveConnection("sample-id")
	require.NoError(t, err)
}

func TestClient_HandleInvitation(t *testing.T) {
	t.Run("test success", func(t *testing.T) {
		c, err := New(&mockprovider.Provider{ServiceValue: &mockprotocol.MockDIDExchangeSvc{},
			WalletValue: &mockwallet.CloseableWallet{}, InboundEndpointValue: "endpoint"})
		require.NoError(t, err)
		inviteReq, err := c.CreateInvitation("agent")
		require.NoError(t, err)
		require.NoError(t, c.HandleInvitation(inviteReq))
	})

	t.Run("test error from handle msg", func(t *testing.T) {
		c, err := New(&mockprovider.Provider{
			ServiceValue: &mockprotocol.MockDIDExchangeSvc{HandleFunc: func(msg dispatcher.DIDCommMsg) error {
				return fmt.Errorf("handle error")
			}},
			WalletValue: &mockwallet.CloseableWallet{}, InboundEndpointValue: "endpoint"})
		require.NoError(t, err)
		inviteReq, err := c.CreateInvitation("agent")
		require.NoError(t, err)
		err = c.HandleInvitation(inviteReq)
		require.Error(t, err)
		require.Contains(t, err.Error(), "handle error")
	})
}

func TestClient_QueryConnectionsByParams(t *testing.T) {
	c, err := New(&mockprovider.Provider{ServiceValue: didexchange.New(nil, &did.MockDIDCreator{}, &mockProvider{})})
	require.NoError(t, err)

	results, err := c.QueryConnections(&QueryConnectionsParams{InvitationKey: "sample-inv-key"})
	require.NoError(t, err)
	require.NotEmpty(t, results)
	for _, result := range results {
		require.NotNil(t, result)
		require.NotNil(t, result.ConnectionID)
	}
}

func TestServiceEvents(t *testing.T) {
	store := &mockstore.MockStore{Store: make(map[string][]byte)}
	didExSvc := didexchange.New(store, &did.MockDIDCreator{}, &mockProvider{})
	c, err := New(&mockprovider.Provider{ServiceValue: didExSvc})
	require.NoError(t, err)
	require.NotNil(t, c)

	id := "valid-thread-id"
	newDidDoc, err := (&did.MockDIDCreator{}).CreateDID()
	require.NoError(t, err)

	request, err := json.Marshal(
		&didexchange.Request{
			Type:  didexchange.ConnectionRequest,
			ID:    id,
			Label: "test",
			Connection: &didexchange.Connection{
				DID:    "B.did@B:A",
				DIDDoc: newDidDoc,
			},
		},
	)
	require.NoError(t, err)

	msg := dispatcher.DIDCommMsg{
		Type:    didexchange.ConnectionRequest,
		Payload: request,
	}

	err = didExSvc.Handle(msg)
	require.NoError(t, err)

	validateState(t, store, id, "responded", 100*time.Millisecond)
}

func validateState(t *testing.T, store storage.Store, id, expected string, timeoutDuration time.Duration) {
	actualState := ""
	timeout := time.After(timeoutDuration)
	for {
		select {
		case <-timeout:
			require.Fail(t, fmt.Sprintf("id=%s expectedState=%s actualState=%s", id, expected, actualState))
			return
		default:
			v, err := store.Get(id)
			actualState = string(v)
			if err != nil || expected != string(v) {
				continue
			}
			return
		}
	}
}

func TestServiceEventError(t *testing.T) {
	store := &mockstore.MockStore{Store: make(map[string][]byte)}
	didExSvc := didexchange.New(store, &did.MockDIDCreator{}, &mockProvider{})

	// register action event on service before client registers it (only one can be registered)
	actionCh := make(chan dispatcher.DIDCommAction)
	err := didExSvc.RegisterActionEvent(actionCh)
	require.NoError(t, err)

	_, err = New(&mockprovider.Provider{ServiceValue: didExSvc})
	require.Error(t, err)
	require.Contains(t, err.Error(), "service event listener startup failed")

	// unregister the action event and client creation shouldn't fail.
	err = didExSvc.UnregisterActionEvent(actionCh)
	require.NoError(t, err)

	c, err := New(&mockprovider.Provider{ServiceValue: didExSvc})
	require.NoError(t, err)
	require.NotNil(t, c)

	// register msg event error
	s := mockprotocol.MockDIDExchangeSvc{
		ProtocolName:        didexchange.DIDExchange,
		RegisterMsgEventErr: errors.New("error"),
	}
	_, err = New(&mockprovider.Provider{ServiceValue: &s})
	require.Error(t, err)
	require.Contains(t, err.Error(), "service event listener startup failed")
}

type mockProvider struct {
}

func (m *mockProvider) OutboundDispatcher() dispatcher.Outbound {
	return &mockdispatcher.MockOutbound{}
}
