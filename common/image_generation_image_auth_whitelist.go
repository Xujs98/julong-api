package common

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// NormalizeImageGenerationLogImageAuthWhitelist validates and canonicalizes
// newline/comma-separated exact IP addresses and domain names.
func NormalizeImageGenerationLogImageAuthWhitelist(raw string) (string, error) {
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		entry, err := normalizeImageGenerationLogImageAuthWhitelistEntry(value)
		if err != nil {
			return "", err
		}
		if _, exists := seen[entry]; exists {
			continue
		}
		seen[entry] = struct{}{}
		normalized = append(normalized, entry)
	}
	return strings.Join(normalized, "\n"), nil
}

func ImageGenerationLogImageAuthWhitelistMatches(raw, clientIP, origin, referer string) bool {
	normalized, err := NormalizeImageGenerationLogImageAuthWhitelist(raw)
	if err != nil || normalized == "" {
		return false
	}

	parsedClientIP := net.ParseIP(strings.TrimSpace(clientIP))
	originHost := requestSourceHostname(origin)
	refererHost := requestSourceHostname(referer)
	for _, entry := range strings.Split(normalized, "\n") {
		if entry == "0.0.0.0" {
			return true
		}
		if whitelistIP := net.ParseIP(entry); whitelistIP != nil {
			if parsedClientIP != nil && parsedClientIP.Equal(whitelistIP) {
				return true
			}
			continue
		}
		if entry == originHost || entry == refererHost {
			return true
		}
	}
	return false
}

func normalizeImageGenerationLogImageAuthWhitelistEntry(raw string) (string, error) {
	entry := strings.TrimSpace(raw)
	if entry == "" {
		return "", fmt.Errorf("白名单包含空值")
	}

	if ip := net.ParseIP(strings.Trim(entry, "[]")); ip != nil {
		return ip.String(), nil
	}

	host := entry
	if strings.Contains(entry, "://") {
		parsed, err := url.Parse(entry)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
			return "", fmt.Errorf("无效的白名单域名或 IP：%s", entry)
		}
		host = parsed.Hostname()
	} else {
		if strings.ContainsAny(entry, "/?#@") {
			return "", fmt.Errorf("无效的白名单域名或 IP：%s", entry)
		}
		if splitHost, _, err := net.SplitHostPort(entry); err == nil {
			host = splitHost
		} else if strings.Contains(entry, ":") {
			return "", fmt.Errorf("无效的白名单域名或 IP：%s", entry)
		}
	}

	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return ip.String(), nil
	}
	if !isValidExactDomain(host) {
		return "", fmt.Errorf("无效的白名单域名或 IP：%s", entry)
	}
	return host, nil
}

func requestSourceHostname(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
}

func isValidExactDomain(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}
