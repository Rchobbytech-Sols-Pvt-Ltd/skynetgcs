//go:build !mock

package activation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jhakrishan20/skynetgcs/internal/config"
)

var (
	ErrEmptyKey          = errors.New("Please enter an activation key.")
	ErrInvalidKey        = errors.New("Invalid activation key.")
	ErrKeyRevoked        = errors.New("This key has been revoked. Please contact support.")
	ErrKeyExpired        = errors.New("This activation key has expired.")
	ErrKeyBoundElsewhere = errors.New("This key is already activated on another machine.")

	ErrNoInternet        = errors.New("No internet connection. Check your network and try again.")
	ErrTimeout           = errors.New("The activation server took too long to respond. Please try again.")
	ErrServerUnreachable = errors.New("Could not reach the activation server. Please try again later.")

	ErrServerMisconfig = errors.New("Activation service is unavailable. Please contact support.")
	ErrServerError     = errors.New("The activation server returned an error. Please try again later.")
	ErrBadResponse     = errors.New("Received an unexpected response from the server. Please try again.")

	ErrMachineID = errors.New("Could not identify this machine. Try restarting the app.")
	ErrUnknown   = errors.New("Activation failed. Please try again.")
)

type activateRequest struct {
	Key       string `json:"key"`
	MachineID string `json:"machine_id"`
}

type activateResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func Activate(key string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, ErrEmptyKey
	}

	machineID, err := MachineID()
	if err != nil {
		log.Printf("[activation] machine id error: %v", err)
		return false, ErrMachineID
	}

	reqBody, err := json.Marshal(activateRequest{Key: key, MachineID: machineID})
	if err != nil {
		log.Printf("[activation] marshal error: %v", err)
		return false, ErrUnknown
	}

	endpoint := config.SupabaseURL + config.ActivateRoute
	log.Printf("[activation] POST %s body=%s", endpoint, reqBody)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		log.Printf("[activation] build request error: %v", err)
		return false, ErrUnknown
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", config.SupabaseAPIKey)

	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[activation] transport error after %s: %v", time.Since(start), err)
		return false, classifyTransportError(err)
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("[activation] read body error: %v", readErr)
		return false, ErrBadResponse
	}
	log.Printf("[activation] status=%d duration=%s body=%s", resp.StatusCode, time.Since(start), raw)

	// Always prefer the structured response payload over the status code —
	// the Edge Function uses 4xx for legitimate rejection cases like
	// "Invalid key" or "Key revoked", and the message field is what we
	// actually want to surface to the user.
	var out activateResponse
	if err := json.Unmarshal(raw, &out); err == nil && out.Message != "" {
		if out.OK {
			log.Printf("[activation] success")
			return true, nil
		}
		return false, classifyRejection(out.Message)
	}

	// No usable JSON payload — classify by HTTP status.
	switch {
	case resp.StatusCode == 200:
		return false, ErrBadResponse
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return false, ErrServerMisconfig
	case resp.StatusCode == 404:
		return false, ErrServerMisconfig
	case resp.StatusCode >= 500:
		return false, ErrServerError
	case resp.StatusCode >= 400:
		return false, ErrServerError
	}
	return false, ErrUnknown
}

func classifyTransportError(err error) error {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrNoInternet
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Timeout() {
		return ErrTimeout
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrTimeout
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no such host"),
		strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "no address associated"):
		return ErrNoInternet
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "deadline exceeded"):
		return ErrTimeout
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no route to host"),
		strings.Contains(msg, "tls"),
		strings.Contains(msg, "certificate"):
		return ErrServerUnreachable
	}
	return ErrServerUnreachable
}

func classifyRejection(serverMsg string) error {
	msg := strings.ToLower(serverMsg)
	switch {
	case msg == "":
		return ErrUnknown
	case strings.Contains(msg, "invalid key"),
		strings.Contains(msg, "not found"),
		strings.Contains(msg, "missing"):
		return ErrInvalidKey
	case strings.Contains(msg, "revoked"):
		return ErrKeyRevoked
	case strings.Contains(msg, "expired"):
		return ErrKeyExpired
	case strings.Contains(msg, "another machine"),
		strings.Contains(msg, "already used"),
		strings.Contains(msg, "already activated"):
		return ErrKeyBoundElsewhere
	}
	return ErrUnknown
}
