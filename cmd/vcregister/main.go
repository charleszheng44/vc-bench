package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/charleszheng44/vc-bench/pkg/vcregister"
)

var (
	metaKbCfg string
	tcKbCfg   string
)

func init() {
	flag.StringVar(&metaKbCfg, "metakbcfg", "",
		"path to the kubeconfig file of the meta cluster.")
	flag.StringVar(&tcKbCfg, "tckbcfg", "",
		"path to the kubeconfig file of the tenant master management cluster.")
	flag.Parse()
}

func main() {
	vcr, err := vcregister.New(metaKbCfg, tcKbCfg)
	if err != nil {
		log.Fatalf("fail to create VirtualclusterRegister: %s", err)
	}

	sigStop := make(chan os.Signal)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)
	stopChan := make(chan struct{})

	go func() {
		signal := <-sigStop
		log.Printf("receive signal(%s), will terminate", signal)
		close(stopChan)
	}()

	log.Print("starting Virtualcluster register")
	if err := vcr.Start(stopChan); err != nil {
		log.Fatalf("fail to run VirtualclusterRegister: %s", err)
	}
}
