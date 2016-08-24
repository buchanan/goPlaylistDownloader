package plexDownloader

import (
	"io"
	"fmt"
	"time"
)

type PipeViewer struct {
	io.Reader
	AmountRead float64
	Total float64
	count float64
	timer time.Time
}

func (pv *PipeViewer) Read(p []byte) (int, error) {
	n, err := pv.Reader.Read(p)
	pv.AmountRead += float64(n)
	pv.count += float64(n)
	
	if since := time.Since(pv.timer); since > time.Second {
		var complete float64 = (pv.AmountRead/pv.Total)*100
		var seconds float64 = float64(since)/1000000000
		var Mbps float64 = (pv.count/seconds)/1048576
		
		pv.count = 0
		pv.timer = time.Now()
		
		if err == nil {
			fmt.Printf("\r\033[K%.2f percent complete %.2f Mbps", complete, Mbps)
		}
	}
	
	return n, err
}