package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fr13n8/icmping"
)

func main() {
	ttl := flag.Int("l", 64, "TTL")
	interval := flag.Duration("i", time.Second, "")
	size := flag.Int("s", 32, "")
	flag.Parse()

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
