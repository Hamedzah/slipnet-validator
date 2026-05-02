package main

import (
	"crypto/tls"
	"fmt"
	"net"
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
	// قالب: slipnet-enc://server:port?param=value
	server, port, err := parseSlipnetURL(rawURL)
	if err != nil {
		return cfg
	}
	address := fmt.Sprintf("%s:%d", server, port)

	// --- مرحله 2: Crypto Handshake (TLS) ---
	tlsStart := time.Now()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", address, &tls.Config{
		InsecureSkipVerify: true, // فقط برای تست، در تولید حتماً اعتبارسنجی کنید
	})
	if err != nil {
		return cfg
	}
	cfg.TLSStatus = true
	tlsLatency := time.Since(tlsStart)
	conn.Close()

	// --- مرحله 3: DOIP Query Response (Latency واقعی) ---
	// ارسال یک درخواست ساده برای سنجش تأخیر واقعی
	latencyStart := time.Now()
	conn, err = net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return cfg
	}
	// ارسال یک پکت تست ساده (مثلاً یک درخوات HTTP ساده)
	_, err = conn.Write([]byte("GET / HTTP/1.0\r\n\r\n"))
	if err != nil {
		conn.Close()
		return cfg
	}
	// خواندن پاسخ (فقط 1KB برای بررسی پاسخ)
	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	conn.Close()
	if err != nil {
		return cfg
	}
	latency := time.Since(latencyStart)

	// --- مرحله 4: Payload Delivery Verification (ثبات اتصال) ---
	steadinessStart := time.Now()
	conn, err = net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return cfg
	}
	// ارسال یک Payload تست بزرگ‌تر (مثلاً 5KB)
	testPayload := make([]byte, 5120) // 5KB
	for i := range testPayload {
		testPayload[i] = byte(i % 256)
	}
	_, err = conn.Write(testPayload)
	if err != nil {
		conn.Close()
		return cfg
	}
	// خواندن پاسخ
	_, err = conn.Read(buf)
	conn.Close()
	if err == nil && time.Since(steadinessStart) < 10*time.Second {
		cfg.Steadiness = true
	}

	// --- مرحله 5: جمع‌بندی ---
	cfg.HealthCheck = true
	cfg.Latency = latency + tlsLatency // زمان کل
	cfg.Valid = cfg.TLSStatus && cfg.Steadiness && (cfg.Latency < 2000*time.Millisecond) // حداکثر 2 ثانیه

	return cfg
}

// parseSlipnetURL اطلاعات سرور و پورت را از URL استخراج می‌کند
func parseSlipnetURL(rawURL string) (string, int, error) {
	// حذف پروتکل
	if len(rawURL) < 12 || rawURL[:12] != "slipnet-enc:" {
		return "", 0, fmt.Errorf("invalid protocol")
	}
	rest := rawURL[12:]
	// حذف //
	if len(rest) >= 2 && rest[:2] == "//" {
		rest = rest[2:]
	}
	// پیدا کردن سرور و پورت
	server := ""
	port := 443 // پورت پیش‌فرض
	for i, ch := range rest {
		if ch == ':' {
			server = rest[:i]
			portPart := rest[i+1:]
			// پیدا کردن پایان پورت (قبل از ? یا / یا پایان رشته)
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
