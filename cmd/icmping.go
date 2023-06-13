package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/fr13n8/icmping"
)

func main() {
	ttl := flag.Int("l", 128, "TTL")
	interval := flag.Duration("i", time.Second, "")
	size := flag.Int("s", 32, "")
	flag.Parse()

	host := flag.Arg(0)
	if len(host) == 0 {
		fmt.Println("host is empty")
	}

	pinger, err := icmping.NewPinger(
		host,
		icmping.WithInterval(*interval),
		icmping.WithTTl(*ttl),
		icmping.WithSize(*size),
	)
	if err != nil {
		fmt.Println(err)
	}

	err = pinger.RunPing()
	if err != nil {
		fmt.Println(err)
	}
}
