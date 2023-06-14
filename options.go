package icmping

import (
	"math"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	ipv4Map   = map[string]string{"icmp": "ip4:icmp", "udp": "udp4"}
	ipv6Map   = map[string]string{"icmp": "ip6:ipv6-icmp", "udp": "udp6"}
	protocols = struct {
		icmp string
		udp  string
	}{
		icmp: "icmp",
		udp:  "udp",
	}
)

type OptionsFunc func(*Options)

type Options struct {
	id  int
	seq int

	// Number of packets sent
	pktsSent int
	// Number of packets received
	pktsRecv int

	// target ip address
	ipaddr *net.IPAddr

	interval time.Duration
	timeout  time.Duration

	rtts   []time.Duration
	minRtt time.Duration
	maxRtt time.Duration

	// is ipv4 or ipv6
	ipv4 bool
	// time to live
	ttl int
	// size of payload
	size int
	// "icmp" or "udp".
	protocol string
	// target address
	address string
	// listen address
	listenAddr string

	statsLock sync.RWMutex
	quit      chan struct{}
}

func defaultOptions() *Options {
	var protocol string
	osType := runtime.GOOS
	switch osType {
	case "windows":
		protocol = protocols.icmp
	case "darwin":
	case "linux":
		protocol = protocols.udp
	default:
		protocol = protocols.icmp
	}

	return &Options{
		// The ICMP ID of the ping utility is defined equal to the process ID which is generated when ping is started.
		id:         os.Getpid() & 0xffff,
		seq:        0,
		interval:   1,
		timeout:    time.Duration(math.MaxInt64),
		protocol:   protocol,
		listenAddr: "0.0.0.0",
		ipv4:       false,
		size:       24,
		quit:       make(chan struct{}),
		pktsSent:   0,
		pktsRecv:   0,
		rtts:       []time.Duration{},
	}
}

func WithInterval(i time.Duration) OptionsFunc {
	return func(o *Options) {
		o.interval = i
	}
}

func WithTTl(i int) OptionsFunc {
	return func(o *Options) {
		o.ttl = i
	}
}

func WithSize(s int) OptionsFunc {
	return func(o *Options) {
		o.size = s
	}
}
