package amt

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Discovered is one host found during a subnet scan.
type Discovered struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	TLS    bool   `json:"tls"`
	Server string `json:"server"` // HTTP Server header, if any
	IsAMT  bool   `json:"isAmt"`  // Server header looks like Intel AMT
}

// maxScanHosts caps a single scan so a typo (e.g. /8) can't fan out forever.
const maxScanHosts = 8192

// Scan probes every address in a CIDR for an open AMT port and reports those
// that respond, flagging hosts whose HTTP Server header identifies as Intel AMT.
func Scan(ctx context.Context, cidr string, port int, useTLS bool, timeout time.Duration) ([]Discovered, error) {
	if port == 0 {
		if useTLS {
			port = DefaultTLSPort
		} else {
			port = DefaultPort
		}
	}
	if timeout == 0 {
		timeout = 800 * time.Millisecond
	}

	ips, err := expandCIDR(cidr)
	if err != nil {
		return nil, err
	}
	if len(ips) > maxScanHosts {
		return nil, fmt.Errorf("scan range too large (%d hosts, max %d)", len(ips), maxScanHosts)
	}

	var (
		mu      sync.Mutex
		results []Discovered
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 128) // bounded concurrency
	)

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(host string) {
			defer wg.Done()
			defer func() { <-sem }()
			if d, ok := probe(ctx, host, port, useTLS, timeout); ok {
				mu.Lock()
				results = append(results, d)
				mu.Unlock()
			}
		}(ip)
	}
	wg.Wait()
	return results, nil
}

// probe attempts a TCP connect and a lightweight HTTP request to read the
// Server header (AMT answers an unauthenticated request with 401 + headers).
func probe(ctx context.Context, host string, port int, useTLS bool, timeout time.Duration) (Discovered, bool) {
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Discovered{}, false
	}
	conn.Close()

	res := Discovered{Host: host, Port: port, TLS: useTLS}

	// Best-effort HTTP probe for the Server header.
	scheme := "http"
	client := &http.Client{Timeout: timeout}
	if useTLS {
		scheme = "https"
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s://%s/", scheme, addr), nil)
	if resp, err := client.Do(req); err == nil {
		res.Server = resp.Header.Get("Server")
		resp.Body.Close()
	}
	res.IsAMT = strings.Contains(strings.ToLower(res.Server), "active management") ||
		strings.Contains(strings.ToLower(res.Server), "amt")

	return res, true
}

// expandCIDR returns all host addresses in a CIDR block (network/broadcast
// included for small blocks; callers typically connect only to live hosts).
func expandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Allow a bare single IP too.
		if parsed := net.ParseIP(cidr); parsed != nil {
			return []string{parsed.String()}, nil
		}
		return nil, fmt.Errorf("invalid CIDR or IP %q", cidr)
	}

	var ips []string
	for cur := ip.Mask(ipnet.Mask); ipnet.Contains(cur); cur = nextIP(cur) {
		ips = append(ips, cur.String())
		if len(ips) > maxScanHosts {
			break
		}
	}
	// Drop network & broadcast for blocks larger than a /31.
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips, nil
}

func nextIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] != 0 {
			break
		}
	}
	return out
}
