package icmping

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type Packet struct {
	nBytes int
	msg    []byte
	TTL    int
	seq    int
	rtt    time.Duration
}

type Pinger struct {
	*Options
}

func NewPinger(address string, options ...OptionsFunc) (*Pinger, error) {
	opts := defaultOptions()
	opts.address = address
	for _, fn := range options {
		fn(opts)
	}
	pinger := &Pinger{opts}

	return pinger, pinger.resolve()
}

func (p *Pinger) resolve() error {
	if len(p.address) == 0 {
		return errors.New("empty address")
	}

	ipaddr, err := net.ResolveIPAddr("ip", p.address)
	if err != nil {
		return err
	}

	p.ipv4 = isIPv4(ipaddr.IP)
	p.ipaddr = ipaddr

	return nil
}

func (p *Pinger) RunPing() error {
	conn, err := p.Listen()
	if err != nil {
		return err
	}

	if err := conn.IPv4PacketConn().SetTTL(p.ttl); err != nil {
		return err
	}
	conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)

	recv := make(chan *Packet, 5)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := p.Ping(conn, recv); err != nil {
			fmt.Println("ERROR: ", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := p.RecvICMPPacket(conn, recv); err != nil {
			fmt.Println("ERROR: ", err)
		}
	}()

	wg.Wait()

	return nil
}

func (p *Pinger) Ping(conn *icmp.PacketConn, recv <-chan *Packet) error {
	interval := time.NewTicker(p.interval)
	timeout := time.NewTicker(p.timeout)

	if err := p.SendICMPPacket(conn); err != nil {
		return err
	}

	for {
		select {
		case <-interval.C:
			if err := p.SendICMPPacket(conn); err != nil {
				fmt.Println("ERROR: ", err)
			}
		case <-timeout.C:
			fmt.Println("TIMEOUT")
			return nil
		case r := <-recv:
			fmt.Printf("Echo reply from %s: bytes=%d time=%v ttl=%d icmp_seq=%d\n", p.ipaddr.IP.String(), r.nBytes, r.rtt, r.TTL, r.seq)
		}
	}
}

func (p *Pinger) SendICMPPacket(conn *icmp.PacketConn) error {
	uuidEncoded, _ := uuid.UUID{}.MarshalBinary()
	bodyBytes := append(timeToBytes(time.Now()), uuidEncoded...)
	if remainSize := p.size - 24; remainSize > 0 {
		bodyBytes = append(bodyBytes, bytes.Repeat([]byte{1}, remainSize)...)
	}

	body := &icmp.Echo{
		ID:   p.id,
		Seq:  p.seq,
		Data: bodyBytes,
	}
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	for {
		if _, err = conn.WriteTo(msgBytes, p.ipaddr); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
			return err
		}

		p.seq++
		break
	}

	return nil
}

func (p *Pinger) RecvICMPPacket(conn *icmp.PacketConn, recv chan<- *Packet) error {
	proto := ipv4.ICMPTypeEchoReply.Protocol()
	if !p.ipv4 {
		proto = ipv6.ICMPTypeEchoReply.Protocol()
	}

	for {
		recvBytes := make([]byte, p.size+28)
		n, _, err := conn.ReadFrom(recvBytes)
		if err != nil {
			return err
		}

		recvMsg, err := icmp.ParseMessage(proto, recvBytes[:n])
		if err != nil {
			return err
		}

		switch recvMsg.Body.(type) {
		case *icmp.Echo:
			echoReply, _ := recvMsg.Body.(*icmp.Echo)

			TTL, err := conn.IPv4PacketConn().TTL()
			if err != nil {
				return err
			}
			receivedAt := time.Now()
			timestamp := bytesToTime(echoReply.Data[:8])

			recv <- &Packet{
				nBytes: len(echoReply.Data),
				msg:    recvBytes,
				TTL:    TTL,
				seq:    echoReply.Seq,
				rtt:    receivedAt.Sub(timestamp),
			}
		default:
			fmt.Println("not an echo reply", recvMsg)
		}
	}
}

func (p *Pinger) Listen() (*icmp.PacketConn, error) {
	var (
		conn *icmp.PacketConn
		err  error
	)
	conn, err = icmp.ListenPacket(ipv4Map[p.protocol], p.listen)
	if !p.ipv4 {
		conn, err = icmp.ListenPacket(ipv6Map[p.protocol], p.listen)
	}
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (p *Pinger) GetIpAddr() *net.IPAddr {
	return p.ipaddr
}

func (p *Pinger) GetIPv4Status() bool {
	return p.ipv4
}
