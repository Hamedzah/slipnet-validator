package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"
)

// CheckerValidateSlipnetConfig اعتبارسنجی عمیق و چندلایه
func CheckerValidateSlipnetConfig(rawURL string) Config {
	cfg := Config{
		RawURL:      rawURL,
		HealthCheck: false,
		TLSStatus:   false,
		Steadiness:  false,
		Valid:       false,
	}

	// --- مرحله 1: استخراج اطلاعات از URL ---
	server, port, err := parseSlipnetURL(rawURL)
	if err != nil {
		return cfg
	}
	address := fmt.Sprintf("%s:%d", server, port)

	// --- مرحله 2: Crypto Handshake (TLS) ---
	tlsStart := time.Now()
	tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return cfg
	}
	cfg.TLSStatus = true
	tlsLatency := time.Since(tlsStart)
	tlsConn.Close()

	// --- مرحله 3: TCP Connection for Latency ---
	latencyStart := time.Now()
	tcpConn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return cfg
	}
	// ارسال یک پکت ساده
	_, err = tcpConn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
	if err != nil {
		tcpConn.Close()
		return cfg
	}
	buf := make([]byte, 1024)
	_, err = tcpConn.Read(buf)
	tcpConn.Close()
	if err != nil {
		return cfg
	}
	latency := time.Since(latencyStart)

	// --- مرحله 4: Steadiness Test با Payload بزرگتر ---
	steadinessStart := time.Now()
	steadyConn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return cfg
	}
	testPayload := make([]byte, 5120) // 5KB
	for i := range testPayload {
		testPayload[i] = byte(i % 256)
	}
	_, err = steadyConn.Write(testPayload)
	if err != nil {
		steadyConn.Close()
		return cfg
	}
	_, err = steadyConn.Read(buf)
	steadyConn.Close()
	if err == nil && time.Since(steadinessStart) < 10*time.Second {
		cfg.Steadiness = true
	}

	// --- مرحله 5: جمع‌بندی ---
	cfg.HealthCheck = true
	cfg.Latency = latency + tlsLatency
	cfg.Valid = cfg.TLSStatus && cfg.Steadiness && (cfg.Latency < 2000*time.Millisecond)

	return cfg
}

// parseSlipnetURL اطلاعات سرور و پورت را از URL استخراج می‌کند
func parseSlipnetURL(rawURL string) (string, int, error) {
	if len(rawURL) < 12 || rawURL[:12] != "slipnet-enc:" {
		return "", 0, fmt.Errorf("invalid protocol")
	}
	rest := rawURL[12:]
	if len(rest) >= 2 && rest[:2] == "//" {
		rest = rest[2:]
	}
	server := ""
	port := 443
	for i, ch := range rest {
		if ch == ':' {
			server = rest[:i]
			portPart := rest[i+1:]
			for j, pc := range portPart {
				if pc == '?' || pc == '/' {
					portStr := portPart[:j]
					if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
						port = p
					}
					break
				} else if j == len(portPart)-1 {
					if p, err := strconv.Atoi(portPart); err == nil && p > 0 && p < 65536 {
						port = p
					}
				}
			}
			break
		}
	}
	if server == "" {
		return "", 0, fmt.Errorf("no server found")
	}
	return server, port, nil
}
