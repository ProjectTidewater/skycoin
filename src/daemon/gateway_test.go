package daemon

import (
	"errors"
	"testing"

	"github.com/skycoin/skycoin/src/daemon/strand"
	"github.com/skycoin/skycoin/src/wallet"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/visor"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/coin"
	"github.com/skycoin/skycoin/src/testutil"
)

func TestFbyAddresses(t *testing.T) {
	uxs := make(coin.UxArray, 5)
	addrs := make([]cipher.Address, 5)
	for i := 0; i < 5; i++ {
		addrs[i] = testutil.MakeAddress()
		uxs[i] = coin.UxOut{
			Body: coin.UxBody{
				Address: addrs[i],
			},
		}
	}

	tests := []struct {
		name    string
		addrs   []string
		outputs []coin.UxOut
		want    []coin.UxOut
	}{
		// TODO: Add test cases.
		{
			"filter with one address",
			[]string{addrs[0].String()},
			uxs[:2],
			uxs[:1],
		},
		{
			"filter with multiple addresses",
			[]string{addrs[0].String(), addrs[1].String()},
			uxs[:3],
			uxs[:2],
		},
	}
	for _, tt := range tests {
		// fmt.Printf("want:%+v\n", tt.want)
		outs := FbyAddresses(tt.addrs)(tt.outputs)
		require.Equal(t, outs, coin.UxArray(tt.want))
	}
}

func TestFbyHashes(t *testing.T) {
	uxs := make(coin.UxArray, 5)
	addrs := make([]cipher.Address, 5)
	for i := 0; i < 5; i++ {
		addrs[i] = testutil.MakeAddress()
		uxs[i] = coin.UxOut{
			Body: coin.UxBody{
				Address: addrs[i],
			},
		}
	}

	type args struct {
		hashes []string
	}
	tests := []struct {
		name    string
		hashes  []string
		outputs coin.UxArray
		want    coin.UxArray
	}{
		// TODO: Add test cases.
		{
			"filter with one hash",
			[]string{uxs[0].Hash().Hex()},
			uxs[:2],
			uxs[:1],
		},
		{
			"filter with multiple hash",
			[]string{uxs[0].Hash().Hex(), uxs[1].Hash().Hex()},
			uxs[:3],
			uxs[:2],
		},
	}
	for _, tt := range tests {
		outs := FbyHashes(tt.hashes)(tt.outputs)
		require.Equal(t, outs, coin.UxArray(tt.want))
	}
}

func TestGateway_GetWallet(t *testing.T) {
	tests := []struct {
		name             string
		disableWalletAPI bool
		walletId         string
		wallet           wallet.Wallet
		getWalletError   error
		err              error
	}{
		{
			name:             "wallet api disabled",
			disableWalletAPI: true,
			err:              wallet.ErrWalletApiDisabled,
		},
		{
			name:             "getWalletError",
			disableWalletAPI: false,
			walletId:         "walletId",
			getWalletError:   errors.New("getWalletError"),
			err:              errors.New("getWalletError"),
		},
		{
			name:             "OK",
			disableWalletAPI: false,
			walletId:         "walletId",
			wallet:           wallet.Wallet{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vrpc := visor.NewRPCIfaceMock()
			gw := &Gateway{
				Config: GatewayConfig{
					DisableWalletAPI: tc.disableWalletAPI,
				},
				drpc:     RPC{},
				vrpc:     vrpc,
				requests: make(chan strand.Request, 32),
				quit:     make(chan struct{}),
			}
			go func() {
				select {
				case req := <-gw.requests:
					req.Func()
				}
			}()

			vrpc.On("GetWallet", tc.walletId).Return(tc.wallet, tc.getWalletError)
			w, err := gw.GetWallet(tc.walletId)
			if err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.Equal(t, tc.wallet, w)
		})
	}
}

func TestGateway_GetWallets(t *testing.T) {
	tests := []struct {
		name             string
		disableWalletAPI bool
		wallets          wallet.Wallets
		getWalletError   error
		err              error
	}{
		{
			name:             "wallet api disabled",
			disableWalletAPI: true,
			err:              wallet.ErrWalletApiDisabled,
		},
		{
			name:             "OK",
			disableWalletAPI: false,
			wallets:          wallet.Wallets{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			vrpc := visor.NewRPCIfaceMock()
			gw := &Gateway{
				Config: GatewayConfig{
					DisableWalletAPI: tc.disableWalletAPI,
				},
				drpc:     RPC{},
				vrpc:     vrpc,
				requests: make(chan strand.Request, 32),
				quit:     make(chan struct{}),
			}
			go func() {
				select {
				case req := <-gw.requests:
					req.Func()
				}
			}()

			vrpc.On("GetWallets").Return(tc.wallets)
			w, err := gw.GetWallets()
			if err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.Equal(t, tc.wallets, w)
		})
	}
}

//func TestGateway_GetWalletUnconfirmedTxns(t *testing.T) {
//	tests := []struct {
//		name             string
//		disableWalletAPI bool
//		walletId         string
//		addrs            []cipher.Address
//		wallets          wallet.Wallets
//		getWalletError   error
//		err              error
//	}{
//		{
//			name:             "wallet api disabled",
//			disableWalletAPI: true,
//			err:              wallet.ErrWalletApiDisabled,
//		},
//		{
//			name:             "OK",
//			disableWalletAPI: false,
//			wallets:          wallet.Wallets{},
//		},
//	}
//
//	for _, tc := range tests {
//		t.Run(tc.name, func(t *testing.T) {
//			vrpc := visor.NewRPCIfaceMock()
//			v := visor.NewVisorerMock()
//			gw := &Gateway{
//				Config: GatewayConfig{
//					DisableWalletAPI: tc.disableWalletAPI,
//				},
//				drpc:     RPC{},
//				vrpc:     vrpc,
//				v:        v,
//				requests: make(chan strand.Request, 32),
//				quit:     make(chan struct{}),
//			}
//			go func() {
//				select {
//				case req := <-gw.requests:
//					req.Func()
//				}
//			}()
//
//			vrpc.On("GetWalletAddresses", tc.walletId).Return(tc.addrs)
//			w, err := gw.GetWalletUnconfirmedTxns(tc.walletId)
//			if err != nil {
//				require.Equal(t, tc.err, err)
//				return
//			}
//			require.Equal(t, tc.wallets, w)
//		})
//	}
//}
