package ecpay

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/st-ember/microtip/internal/app/port/hash"
)

type ECPayHasher struct {
	hashKey string
	hashIV  string
}

func NewECPayHasher(hashKey string, hashIV string) hash.Hasher {
	return &ECPayHasher{
		hashKey: hashKey,
		hashIV:  hashIV,
	}
}

// ECPayUrlEncode encodes strings to ECPay-compliant form.
func ECPayUrlEncode(s string) string {
	// Standard url escape
	s = url.QueryEscape(s)

	// Convert URL-encoded hex characters (%XX) to lowercase
	s = strings.ToLower(s)

	// Replace %2d with -, %5f with _, %2e with ., %21 with !, %2a with *, %28 with (, %29 with )
	s = strings.ReplaceAll(s, "%2d", "-")
	s = strings.ReplaceAll(s, "%5f", "_")
	s = strings.ReplaceAll(s, "%2e", ".")
	s = strings.ReplaceAll(s, "%21", "!")
	s = strings.ReplaceAll(s, "%2a", "*")
	s = strings.ReplaceAll(s, "%28", "(")
	s = strings.ReplaceAll(s, "%29", ")")

	// Ensure spaces are represented as '+' (Go's QueryEscape already does this, but keeping %20 replacement for safety)
	s = strings.ReplaceAll(s, "%20", "+")

	// Replace ~ with %7e to match traditional ECPay requirements (.NET HttpUtility.UrlEncode behavior)
	s = strings.ReplaceAll(s, "~", "%7e")

	return s
}

func (h *ECPayHasher) GenerateCheckMacVal(params map[string]string) (string, error) {
	// 1. Get all keys, sort them alphabetically, excluding CheckMacValue itself
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "CheckMacValue" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. Concat key=value with &
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	joined := strings.Join(parts, "&")

	// 3. Add HashKey at the start and HashIV at the end
	rawString := fmt.Sprintf("HashKey=%s&%s&HashIV=%s", h.hashKey, joined, h.hashIV)

	// 4. URL encode
	encodedString := ECPayUrlEncode(rawString)

	// 5. Encrypt (default to SHA256 unless EncryptType is "0" for MD5)
	encryptType := "1" // SHA256 by default
	if val, ok := params["EncryptType"]; ok {
		encryptType = val
	}

	var hashResult string
	if encryptType == "0" {
		sum := md5.Sum([]byte(encodedString))
		hashResult = fmt.Sprintf("%x", sum)
	} else {
		sum := sha256.Sum256([]byte(encodedString))
		hashResult = fmt.Sprintf("%x", sum)
	}

	// 6. Convert to uppercase
	return strings.ToUpper(hashResult), nil
}

func (h *ECPayHasher) ValidateCheckMacVal(params map[string]string, incomingMAC string) (bool, error) {
	generated, err := h.GenerateCheckMacVal(params)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(generated, incomingMAC), nil
}
