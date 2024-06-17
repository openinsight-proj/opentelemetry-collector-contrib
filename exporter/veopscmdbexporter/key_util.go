package veopscmdbexporter

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func buildAPIKey(path, key, secret string, params map[string]interface{}) map[string]interface{} {
	// Collect values from params and construct the _secret
	var keys []string
	var values []string

	for k, v := range params {
		if k != "_key" && k != "_secret" {
			keys = append(keys, k)
			values = append(values, fmt.Sprint(v))
		}
	}

	sort.Strings(keys)

	var valueStr string
	for _, k := range keys {
		valueStr += fmt.Sprint(params[k])
	}

	secretStr := path + secret + valueStr

	// Hash the _secret using SHA-1
	hash := sha1.New()
	hash.Write([]byte(secretStr))
	sha1Hash := hex.EncodeToString(hash.Sum(nil))

	// Update params with _key and _secret
	params["_key"] = key
	params["_secret"] = sha1Hash

	return params
}

func makeRequest(method, url string, payload map[string]interface{}) error {
	jsonPayload := toJSON(payload)

	req, err := http.NewRequest(method, url, strings.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read and handle response
	// Example assumes JSON response handling
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}
	return nil
}

func toJSON(payload map[string]interface{}) string {
	jsonStr := "{"
	for key, value := range payload {
		jsonStr += fmt.Sprintf(`"%s":"%v",`, key, value)
	}
	jsonStr = strings.TrimSuffix(jsonStr, ",")
	jsonStr += "}"
	return jsonStr
}

func urlPath(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return parsedURL.Path
}
