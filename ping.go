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
	ttl    int
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

func (p *Pinger) Stop() {
	p.statsLock.Lock()
	defer p.statsLock.Unlock()

	p.quit <- struct{}{}
}

func (p *Pinger) resolve() error {
	if len(p.address) == 0 {
		return errors.New("empty address")
	}

	ipaddr, err := net.ResolveIPAddr("ip", p.address)
	if err != nil {
		// if _, ok := err.(*net.DNSError); ok {
		// 	// TODO
		// }
		return err
	}

	p.ipv4 = isIPv4(ipaddr.IP)
	p.ipaddr = ipaddr

	return nil
}

func (p *Pinger) RunPing() error {
	defer p.PrintStatistics()
	conn, err := p.listen()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.IPv4PacketConn().SetTTL(p.ttl); err != nil {
		return err
	}
	conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)

	recv := make(chan *Packet, 1)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer func() {
			wg.Done()
			p.Stop()
		}()
		if err := p.ping(conn, recv); err != nil {
			fmt.Println("ERROR: ", err)
		}
	}()

	go func() {
		defer func() {
			wg.Done()
			p.Stop()
		}()
		if err := p.recvICMPPacket(conn, recv); err != nil {
			fmt.Println("ERROR: ", err)
		}
	}()

	wg.Wait()

	return nil
}

func (p *Pinger) ping(conn *icmp.PacketConn, recv <-chan *Packet) error {
	fmt.Printf("Pinging %s (%s) with %d bytes of payload:\n\n", p.address, p.ipaddr.IP.String(), p.size)
	interval := time.NewTicker(p.interval)
	timeout := time.NewTicker(p.timeout)
	defer func() {
		interval.Stop()
		timeout.Stop()
	}()

	if err := p.sendICMPPacket(conn); err != nil {
		return err
	}

	for {
		select {
		case <-interval.C:
			if err := p.sendICMPPacket(conn); err != nil {
				fmt.Println("ERROR: ", err)
			}
		case <-p.quit:
			return nil
		case <-timeout.C:
			return nil
		case r := <-recv:
			fmt.Printf("Echo reply from %s: bytes=%d time=%v ttl=%d icmp_seq=%d\n", p.ipaddr.IP.String(), r.nBytes, r.rtt, r.ttl, r.seq)
		default:
			if p.pktsRecv == p.pktCount && p.pktCount > 0 {
				return nil
			}
		}
	}
}

func (p *Pinger) PrintStatistics() {
	loss := float64(p.pktsSent-p.pktsRecv) / float64(p.pktsSent) * 100
	avg := getAvg(p.rtts)
	mdev := calculateMdev(p.rtts)
	fmt.Printf("\n--- %s ping packets statistics ---\n", p.address)
	fmt.Printf("%d packets transmitted, %d received, (%d%%)%d packet loss\n", p.pktsSent, p.pktsRecv, int(loss), p.pktsSent-p.pktsRecv)
	fmt.Printf("--- %s ping rtt statistics ---\n", p.address)
	fmt.Printf("min %v, max %v, avg %v, mdev %v\n", p.minRtt, p.maxRtt, time.Duration(avg), time.Duration(mdev))
}

func (p *Pinger) sendICMPPacket(conn *icmp.PacketConn) error {
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
		if err := conn.SetWriteDeadline(time.Now().Add(4000 * time.Millisecond)); err != nil {
			return err
		}

		p.pktsSent++
		if _, err = conn.WriteTo(msgBytes, p.ipaddr); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Timeout() {
					fmt.Println("Sent request timed out.")
					continue
				}
				if neterr.Err == syscall.ENOBUFS {
					fmt.Println("ENOBUFS")
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

func (p *Pinger) recvICMPPacket(conn *icmp.PacketConn, recv chan<- *Packet) error {
	proto := ipv4.ICMPTypeEchoReply.Protocol()
	if !p.ipv4 {
		proto = ipv6.ICMPTypeEchoReply.Protocol()
	}

	for {
		select {
		case <-p.quit:
			return nil
		default:
			recvBytes := make([]byte, p.size+28)

			if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100)); err != nil {
				return err
			}
			n, _, err := conn.ReadFrom(recvBytes)
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						// Request timed out.
						continue
					}
					p.Stop()
					return err
				}
			}

			recvMsg, err := icmp.ParseMessage(proto, recvBytes[:n])
			if err != nil {
				return err
			}

			if echoReply, ok := recvMsg.Body.(*icmp.Echo); ok {
				headers, _ := ipv4.ParseHeader(recvBytes)
				// TTL, err := conn.IPv4PacketConn().TTL()
				// if err != nil {
				// 	return err
				// }
				receivedAt := time.Now()
				timestamp := bytesToTime(echoReply.Data[:8])
				rtt := receivedAt.Sub(timestamp)
				pkt := &Packet{
					nBytes: len(echoReply.Data),
					msg:    recvBytes,
					ttl:    headers.TTL, // TODO
					seq:    echoReply.Seq,
					rtt:    rtt,
				}
				recv <- pkt

				p.pktsRecv++
				p.statsUpdate(pkt)
			}
		}
	}
}

func (p *Pinger) statsUpdate(pkt *Packet) {
	p.statsLock.Lock()
	defer p.statsLock.Unlock()

	p.rtts = append(p.rtts, pkt.rtt)
	if p.pktsRecv == 1 || pkt.rtt < p.minRtt {
		p.minRtt = pkt.rtt
	}

	if pkt.rtt > p.maxRtt {
		p.maxRtt = pkt.rtt
	}
}

func (p *Pinger) listen() (*icmp.PacketConn, error) {
	var (
		conn *icmp.PacketConn
		err  error
	)
	conn, err = icmp.ListenPacket(ipv4Map[p.protocol], p.listenAddr)
	if !p.ipv4 {
		conn, err = icmp.ListenPacket(ipv6Map[p.protocol], p.listenAddr)
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
