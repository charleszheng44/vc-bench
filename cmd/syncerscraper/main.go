package main

import (
	"flag"
	"log"
	"time"

	"github.com/charleszheng44/vc-bench/pkg/vcbench"
)

var (
	outDir         string
	syncerAddr     string
	scrapeInterval int
	timeOutMinutes int
)

func init() {
	flag.StringVar(&outDir, "outdir", "", "the output directory of the log file")
	flag.StringVar(&syncerAddr, "synceraddr", "", "the address (host:port) of the syncer ")
	flag.IntVar(&scrapeInterval, "interval", 1, "the scraping interval")
	flag.IntVar(&timeOutMinutes, "timeout", 60, "the scraping interval")
	flag.Parse()
}

func main() {
	log.Printf("will write syncer metrics to %s", outDir)
	go vcbench.ScrapeSyncer(make(chan struct{}), outDir, syncerAddr, scrapeInterval)
	<-time.After(time.Minute * time.Duration(timeOutMinutes))
	log.Print("done")
	return
}
