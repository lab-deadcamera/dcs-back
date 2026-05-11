package studio

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	AssetsRegion  = "ap-southeast-1"
	AssetsService = "ark"
	AssetsVersion = "2024-01-01"
	AssetsHost    = "open.byteplusapi.com"
)

type SignedFetchInput struct {
	AK      string
	SK      string
	Region  string
	Service string
	Action  string
	Version string
	Body    interface{}
	Method  string
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func getDerivedKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte(secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("request"))
	return kSigning
}

func uriEncode(s string, encodeSlash bool) string {
	encoded := ""
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			encoded += string(c)
		} else if c == '/' && !encodeSlash {
			encoded += "/"
		} else {
			encoded += fmt.Sprintf("%%%02X", c)
		}
	}
	return encoded
}

func buildCanonicalQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = uriEncode(k, true) + "=" + uriEncode(params[k], true)
	}
	return strings.Join(pairs, "&")
}

func SignedFetch(input SignedFetchInput) (map[string]interface{}, error) {
	if input.AK == "" || input.SK == "" {
		return nil, fmt.Errorf("AK and SK are both required for signed requests")
	}

	method := input.Method
	if method == "" {
		method = "POST"
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := amzDate[:8]

	canonicalURI := "/"
	queryParams := map[string]string{
		"Action":  input.Action,
		"Version": input.Version,
	}
	canonicalQuery := buildCanonicalQuery(queryParams)

	var bodyStr string
	if input.Body != nil {
		b, err := json.Marshal(input.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyStr = string(b)
	}
	payloadHash := sha256Hex(bodyStr)

	headers := map[string]string{
		"content-type":     "application/json",
		"host":             AssetsHost,
		"x-date":           amzDate,
		"x-content-sha256": payloadHash,
	}

	signedHeadersList := make([]string, 0, len(headers))
	for k := range headers {
		signedHeadersList = append(signedHeadersList, k)
	}
	sort.Strings(signedHeadersList)

	canonicalHeaders := ""
	for _, h := range signedHeadersList {
		canonicalHeaders += h + ":" + headers[h] + "\n"
	}
	signedHeaders := strings.Join(signedHeadersList, ";")

	canonicalRequest := method + "\n" +
		canonicalURI + "\n" +
		canonicalQuery + "\n" +
		canonicalHeaders + "\n" +
		signedHeaders + "\n" +
		payloadHash

	algorithm := "HMAC-SHA256"
	credentialScope := dateStamp + "/" + input.Region + "/" + input.Service + "/request"

	stringToSign := algorithm + "\n" +
		amzDate + "\n" +
		credentialScope + "\n" +
		sha256Hex(canonicalRequest)

	signingKey := getDerivedKey(input.SK, dateStamp, input.Region, input.Service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authorization := algorithm + " Credential=" + input.AK + "/" + credentialScope +
		", SignedHeaders=" + signedHeaders + ", Signature=" + signature

	requestURL := "https://" + AssetsHost + "/?" + canonicalQuery

	req, err := http.NewRequest(method, requestURL, bytes.NewReader([]byte(bodyStr)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", AssetsHost)
	req.Header.Set("X-Date", amzDate)
	req.Header.Set("X-Content-Sha256", payloadHash)
	req.Header.Set("Authorization", authorization)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("signed request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		result = map[string]interface{}{"raw": string(body)}
	}

	if resp.StatusCode >= 400 {
		errMsg := extractAssetsError(result, string(body))
		return nil, fmt.Errorf("BytePlus signed request failed [%d]: %s", resp.StatusCode, errMsg)
	}

	if r, ok := result["Result"]; ok {
		if m, ok := r.(map[string]interface{}); ok {
			return m, nil
		}
	}

	return result, nil
}

func extractAssetsError(result map[string]interface{}, raw string) string {
	if rm, ok := result["ResponseMetadata"].(map[string]interface{}); ok {
		if e, ok := rm["Error"].(map[string]interface{}); ok {
			if msg, ok := e["Message"].(string); ok {
				return msg
			}
		}
	}
	if e, ok := result["Error"].(map[string]interface{}); ok {
		if msg, ok := e["Message"].(string); ok {
			return msg
		}
	}
	if msg, ok := result["message"].(string); ok {
		return msg
	}
	if len(raw) > 400 {
		raw = raw[:400]
	}
	return raw
}
