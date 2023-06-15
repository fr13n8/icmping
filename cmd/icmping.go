package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fr13n8/icmping"
)

var usage = ` 
Usage:
    icmping [-c count] [-i interval] [-t timeout] [-s size] [-l ttl] host

Examples:
    # ping continuously
    icmping 8.8.8.8

    # ping 5 times
    icmping -c 5 8.8.8.8
	
    # ping for 5 seconds
    icmping -t 5s 8.8.8.8

    # ping at 500ms intervals
    icmping -i 500ms 8.8.8.8

    # ping for 5 seconds
    icmping -t 5s 8.8.8.8

    # Set 100-byte payload size
    icmping -s 100 8.8.8.8
`

func main() {
	ttl := flag.Int("l", 64, "TTL")
	count := flag.Int("c", -1, "packets count")
	interval := flag.Duration("i", time.Second, "packets interval")
	timeout := flag.Duration("t", time.Duration(math.MaxInt64), "pinging timeout")
	size := flag.Int("s", 32, "payload size")
	help := flag.Bool("h", false, "help")
	flag.Usage = func() {
		fmt.Print(usage)
	}
	flag.Parse()

	if flag.NArg() == 0 || *help {
		flag.Usage()
		return
	}

	host := flag.Arg(0)
	if len(host) == 0 {
		fmt.Println("host is empty")
		return
	}

	pinger, err := icmping.NewPinger(
		host,
		icmping.WithInterval(*interval),
		icmping.WithTTl(*ttl),
		icmping.WithSize(*size),
		icmping.WithTimeout(*timeout),
		icmping.WithCount(*count),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Shutdown Gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	go func() {
		<-quit
		pinger.Stop()
	}()

	err = pinger.RunPing()
	if err != nil {
		fmt.Println(err)
		return
	}
}
