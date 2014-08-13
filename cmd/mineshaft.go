package main

import (
	"github.com/mattrobenolt/mineshaft/api"
	"github.com/mattrobenolt/mineshaft/carbon"
	"github.com/mattrobenolt/mineshaft/config"
	"github.com/mattrobenolt/mineshaft/pickle"

	"fmt"
	"log"
	"runtime"
)

func printBanner() {
	fmt.Print(`
 ███▄ ▄███▓ ██▓ ███▄    █ ▓█████   ██████  ██░ ██  ▄▄▄        █████▒▄▄▄█████▓
▓██▒▀█▀ ██▒▓██▒ ██ ▀█   █ ▓█   ▀ ▒██    ▒ ▓██░ ██▒▒████▄    ▓██   ▒ ▓  ██▒ ▓▒
▓██    ▓██░▒██▒▓██  ▀█ ██▒▒███   ░ ▓██▄   ▒██▀▀██░▒██  ▀█▄  ▒████ ░ ▒ ▓██░ ▒░
▒██    ▒██ ░██░▓██▒  ▐▌██▒▒▓█  ▄   ▒   ██▒░▓█ ░██ ░██▄▄▄▄██ ░▓█▒  ░ ░ ▓██▓ ░
▒██▒   ░██▒░██░▒██░   ▓██░░▒████▒▒██████▒▒░▓█▒░██▓ ▓█   ▓██▒░▒█░      ▒██▒ ░
░ ▒░   ░  ░░▓  ░ ▒░   ▒ ▒ ░░ ▒░ ░▒ ▒▓▒ ▒ ░ ▒ ░░▒░▒ ▒▒   ▓▒█░ ▒ ░      ▒ ░░
░  ░      ░ ▒ ░░ ░░   ░ ▒░ ░ ░  ░░ ░▒  ░ ░ ▒ ░▒░ ░  ▒   ▒▒ ░ ░          ░
░      ░    ▒ ░   ░   ░ ░    ░   ░  ░  ░   ░  ░░ ░  ░   ▒    ░ ░      ░
       ░    ░           ░    ░  ░      ░   ░  ░  ░      ░  ░

`)
}

func main() {
	printBanner()

	runtime.GOMAXPROCS(runtime.NumCPU())

	conf, err := config.Open()
	if err != nil {
		panic(err)
	}
	log.Println(conf)

	store, err := conf.OpenStore()
	if err != nil {
		panic(err)
	}
	defer store.Close()

	go carbon.ListenAndServe(conf.Carbon.Host+":"+conf.Carbon.Port, store)
	go api.ListenAndServe(conf.Http.Host+":"+conf.Http.Port, store)
	// TODO: add config for the pickle port
	go pickle.ListenAndServe(conf.Http.Host+":2004", store)
	select {}
}
