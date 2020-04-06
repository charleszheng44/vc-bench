package vcbench

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

func ScrapeSyncer(stop <-chan struct{}, outDataDir, syncerAddr string, scrapeInterval int) {
	syncerDataPath := path.Join(outDataDir, fmt.Sprintf("%s.syncer.metrics", outDataDir))
	sdf, err := os.OpenFile(syncerDataPath, os.O_CREATE|os.O_RDWR, 0644)
	defer sdf.Close()
	if err != nil {
		log.Printf("error scrape syncer: %s", err)
		return
	}
	if _, wrtErr := sdf.WriteString(fmt.Sprintf("---------- start at %d ----------\n", time.Now().Unix())); wrtErr != nil {
		log.Printf("error scrape syncer: %s", wrtErr)
		return
	}
	for {
		select {
		case <-stop:
			sdf.WriteString(fmt.Sprintf("---------- end at %d ----------\n", time.Now().Unix()))
			return
		default:
			// parse http response
			if !strings.HasPrefix(syncerAddr, "http://") &&
				!strings.HasPrefix(syncerAddr, "https://") {
				syncerAddr = "http://" + syncerAddr
			}
			rep, netErr := http.Get(syncerAddr)
			if netErr != nil {
				log.Printf("error scrape syncer: %s", netErr)
				return
			}
			defer rep.Body.Close()
			scanner := bufio.NewScanner(rep.Body)
			for scanner.Scan() {
				if _, wrtErr := sdf.WriteString(scanner.Text() + "\n"); wrtErr != nil {
					log.Printf("error scrape syncer: %s", wrtErr)
					return
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
			}

			if _, wrtErr := sdf.WriteString(fmt.Sprintf("---------- record at %d ----------\n", time.Now().Unix())); wrtErr != nil {
				log.Printf("error scrape syncer: %s", wrtErr)
				return
			}
			<-time.After(time.Duration(scrapeInterval) * time.Second)
		}
	}
}
