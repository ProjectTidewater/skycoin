package main

import (
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/skycoin/skycoin/src/api"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cli"
	"github.com/skycoin/skycoin/src/coin"
	"github.com/skycoin/skycoin/src/util/droplet"
	"github.com/skycoin/skycoin/src/wallet"
)

func run() error {
	csvFile := flag.String("csv", "", "csv file to load (format: skyaddress,coins). coins should be whole numbers or decimal strings, e.g. 10 or 10.123000")
	walletFile := flag.String("wallet", "", "wallet file")
	rpcAddr := flag.String("rpc-addr", "http://127.0.0.1:6420", "rpc interface address")
	doSend := flag.Bool("send", false, "send the transaction (by default, creates and prints the txn, but does not send it)")

	flag.Parse()

	if *csvFile == "" {
		return errors.New("csv required")
	}
	if *walletFile == "" {
		return errors.New("wallet required")
	}

	wlt, err := wallet.Load(*walletFile)
	if err != nil {
		return err
	}

	if len(wlt.Entries) == 0 {
		return errors.New("Wallet is empty")
	}

	changeAddr := wlt.Entries[0].Address.String()

	f, err := os.Open(*csvFile)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	fields, err := r.ReadAll()
	if err != nil {
		return err
	}

	var sends []cli.SendAmount
	var errs []error
	for _, f := range fields {
		addr := f[0]

		addr = strings.TrimSpace(addr)

		if _, err := cipher.DecodeBase58Address(addr); err != nil {
			err = fmt.Errorf("Invalid address %s: %v", addr, err)
			errs = append(errs, err)
			continue
		}

		coins, err := droplet.FromString(f[1])
		if err != nil {
			err = fmt.Errorf("Invalid amount %s: %v", f[1], err)
			errs = append(errs, err)
			continue
		}

		sends = append(sends, cli.SendAmount{
			Addr:  addr,
			Coins: coins,
		})
	}

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println("ERROR:", err)
		}
		return errs[0]
	}

	c := api.NewClient(*rpcAddr)

	txn, err := cli.CreateRawTxFromWallet(c, *walletFile, changeAddr, sends, nil)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", txn)

	var totalCoins, totalHours uint64
	fmt.Println("Txn.Out:")
	for _, x := range txn.Out {
		coins, err := droplet.ToString(x.Coins)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s Coins:%s Hours:%d\n", x.Address, coins, x.Hours)

		totalHours, err = coin.AddUint64(totalHours, x.Hours)
		if err != nil {
			panic(err)
		}

		totalCoins, err = coin.AddUint64(totalCoins, x.Coins)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Total outputs:", len(txn.Out))
	totalCoinsStr, err := droplet.ToString(totalCoins)
	if err != nil {
		panic(err)
	}
	fmt.Println("Total coins:", totalCoinsStr)
	fmt.Println("Total hours:", totalHours)

	txnStr := hex.EncodeToString(txn.Serialize())
	fmt.Println("rawtx:", txnStr)

	if *doSend {
		txid, err := c.InjectTransaction(txnStr)
		if err != nil {
			return err
		}

		fmt.Println("txid:", txid)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
}
