package icmping

import (
	"math"
	"net"
	"os"
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

	ipaddr   *net.IPAddr
	interval time.Duration
	timeout  time.Duration
	ipv4     bool

	// time to live
	ttl  int
	size int

	// "icmp" or "udp".
	protocol string
	address  string
	listen   string
}

func defaultOptions() *Options {
	return &Options{
		// The ICMP ID of the ping utility is defined equal to the process ID which is generated when ping is started.
		id:       os.Getpid() & 0xffff,
		seq:      0,
		interval: 1,
		timeout:  time.Duration(math.MaxInt64),
		protocol: protocols.icmp,
		listen:   "0.0.0.0",
		ipv4:     false,
		size:     24,
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
