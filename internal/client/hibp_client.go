package client

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HIBPClient is a client for the Have I Been Pwned API.
type HIBPClient struct {
	httpClient *http.Client
}

// NewHIBPClient creates a new HIBPClient.
func NewHIBPClient() *HIBPClient {
	return &HIBPClient{
		httpClient: &http.Client{},
	}
}

// IsPasswordPwned checks if a password has been pwned by checking its SHA-1 hash
// against the HIBP Pwned Passwords API.
// See: https://haveibeenpwned.com/API/v3#PwnedPasswords
func (c *HIBPClient) IsPasswordPwned(password string) (bool, error) {
	// 1. Hash the password using SHA-1
	h := sha1.New()
	if _, err := io.WriteString(h, password); err != nil {
		return false, err
	}
	sha1Hash := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
	prefix := sha1Hash[:5]
	suffix := sha1Hash[5:]

	// 2. Call the HIBP API with the first 5 characters of the hash
	url := fmt.Sprintf("https://api.pwnedpasswords.com/range/%s", prefix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "go-backend-base") // HIBP API requires a User-Agent

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HIBP API returned status: %s", resp.Status)
	}

	// 3. Check if the hash suffix is in the response body
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 2 && parts[0] == suffix {
			return true, nil // Found the pwned password
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil // Password not found in pwned list
}
