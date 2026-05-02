package main

import (
    "regexp"

    "mvdan.cc/xurls/v2"
)

// ExtractSlipnetURLs تمام لینک‌های slipnet-enc:// را از متن استخراج می‌کند
func ExtractSlipnetURLs(content string) []string {
    // ریجکس اختصاصی برای slipnet-enc://
    slipnetRegex := regexp.MustCompile(`slipnet-enc://[A-Za-z0-9+/=]+`)
    matches := slipnetRegex.FindAllString(content, -1)

    // همچنین از xurls برای یافتن URLهای عمومی استفاده می‌کنیم
    rxRelaxed := xurls.Relaxed()
    allURLs := rxRelaxed.FindAllString(content, -1)

    // فیلتر کردن URLهایی که با slipnet-enc شروع می‌شوند
    urlSet := make(map[string]bool)
    for _, u := range matches {
        urlSet[u] = true
    }
    for _, u := range allURLs {
        if len(u) >= 12 && u[:12] == "slipnet-enc:" {
            urlSet[u] = true
        }
    }

    // تبدیل به اسلایس
    result := make([]string, 0, len(urlSet))
    for u := range urlSet {
        result = append(result, u)
    }
    return result
}
