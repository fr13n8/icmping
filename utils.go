package icmping

import (
	"math"
	"net"
	"time"
)

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

// func isIPv6(ip net.IP) bool {
// 	return len(ip) == net.IPv6len
// }

func calculateMdev(rtts []time.Duration) time.Duration {
	sumRttSquared := float64(0)
	sumRtt := float64(0)
	n := float64(len(rtts))

	for _, value := range rtts {
		rtt := value.Seconds()
		sumRttSquared += rtt * rtt
		sumRtt += rtt
	}

	meanRtt := sumRtt / n
	mdev := math.Sqrt(sumRttSquared/n - math.Pow(meanRtt, 2))

	return time.Duration(mdev * float64(time.Second))
}

func getAvg(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sum := time.Duration(0)
	for _, v := range values {
		sum += v
	}

	return sum / time.Duration(len(values))
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}
