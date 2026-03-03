package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"crypto/subtle"
)

// calculateHMACSHA256 calculates HMAC-SHA256 signature
func calculateHMACSHA256(message, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// secureCompare compares two strings in constant time
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
