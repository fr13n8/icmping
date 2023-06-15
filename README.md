# Icmping

A simple ICMP echo (ping) implementation in Go

## Build

```
go build -ldflags "-s -w" .\cmd\icmping.go
```

### Linux

```
sudo setcap cap_net_raw+ep icmping
```

## Usage

```
$ icmping -h
Usage:
    icmping [-c count][-1] [-i interval][1s] [-t timeout][-1s] [-s size][32] [-l ttl][128] host

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
```
