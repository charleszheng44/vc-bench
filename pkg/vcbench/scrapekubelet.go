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

func ScrapeKubelet(stop <-chan struct{}, outDataDir, kubeletAddr string, scrapeInterval int) {
	kubeletDataPath := path.Join(outDataDir, fmt.Sprintf("%s.kubelet.metrics", outDataDir))
	kdf, err := os.OpenFile(kubeletDataPath, os.O_CREATE|os.O_RDWR, 0644)
	defer kdf.Close()
	if err != nil {
		log.Printf("error scrape kubelet: %s", err)
		return
	}
	if _, wrtErr := kdf.WriteString(fmt.Sprintf("---------- start at %d ----------\n", time.Now().Unix())); wrtErr != nil {
		log.Printf("error scrape kubelet : %s", wrtErr)
		return
	}
	for {
		select {
		case <-stop:
			kdf.WriteString(fmt.Sprintf("---------- end at %d ----------\n", time.Now().Unix()))
			return
		default:
			// parse http response
			if !strings.HasPrefix(kubeletAddr, "http://") &&
				!strings.HasPrefix(kubeletAddr, "https://") {
				kubeletAddr = "http://" + kubeletAddr
			}
			rep, netErr := http.Get(kubeletAddr)
			if netErr != nil {
				log.Printf("error scrape kubelet: %s", netErr)
				return
			}
			defer rep.Body.Close()
			scanner := bufio.NewScanner(rep.Body)
			for scanner.Scan() {
				if _, wrtErr := kdf.WriteString(scanner.Text() + "\n"); wrtErr != nil {
					log.Printf("error scrape kubelet: %s", wrtErr)
					return
				}
			}
			if scanErr := scanner.Err(); scanErr != nil {
			}

			if _, wrtErr := kdf.WriteString(fmt.Sprintf("---------- record at %d ----------\n", time.Now().Unix())); wrtErr != nil {
				log.Printf("error scrape kubelet: %s", wrtErr)
				return
			}
			<-time.After(time.Duration(scrapeInterval) * time.Second)
		}
	}
}
