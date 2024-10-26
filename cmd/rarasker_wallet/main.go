package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	w "github.com/elnosh/gonuts/wallet"
)

const minturl = "http://localhost:8080"

func main() {

	log.Println("stating up request for file")

	// Setup wallet
	config := w.Config{
		WalletPath:     ".",
		CurrentMintURL: minturl,
	}
	wallet, err := w.LoadWallet(config)
	if err != nil {
		log.Panicf(`w.LoadWallet(config). %w`, err)
	}

	// mint ecash
	mintQuote, err := wallet.RequestMint(1000)
	if err != nil {
		log.Panicf(`wallet.RequestMint(). %w`, err)
	}

	proofs, err := wallet.MintTokens(mintQuote.Quote)
	if err != nil {
		log.Panicf(`wallet.MintTokens(mintQuote.Quote) %w`, err)
	}

	fmt.Printf("\n proofs %+v \n", proofs)

	// request blob hello world add 50 sats on header cashu
	// Create request
	client := &http.Client{}

	url := "http://localhost:8070/a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Panicf(`http.NewRequest("GET", url, nil) %w`, err)
	}

	token, err := wallet.Send(50, minturl, true)
	if err != nil {
		log.Panicf(`wallet.Send(50, minturl, true ) %w`, err)
	}

	req.Header.Add("cashu", token.ToString())

	res, err := client.Do(req)
	if err != nil {
		log.Panicf(`client.Do(req) %w`, err)
	}

	fmt.Printf("\n res %+v\n", res)

	if res.StatusCode != 200 {
	    fmt.Printf("\n STATUS  CODE %+v\n", res.StatusCode)
	    fmt.Printf("\n YOU ARE FUCKING POOR \n")
		return
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(res.Body)
	if err != nil {
		log.Panic(`buf.ReadFrom(c.Request.Body) %w`, err)
	}

	fmt.Printf("\n BODY: %s\n", buf.Bytes())

	// res.Body.

}
