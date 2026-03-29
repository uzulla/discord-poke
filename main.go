package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type target struct {
	kind string
	id   string
}

type webhookPayload struct {
	Content  string `json:"content"`
	Username string `json:"username,omitempty"`
}

type successResult struct {
	Target    string `json:"target"`
	MessageID string `json:"messageId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	DryRun    bool   `json:"dryRun"`
}

func main() {
	var targetArg string
	var message string
	var senderName string
	var dryRun bool

	flag.StringVar(&targetArg, "target", "", "target in the form discord-channel:<id> or discord-thread:<id>")
	flag.StringVar(&message, "message", "", "message body to post")
	flag.StringVar(&senderName, "sender-name", "", "optional sender name shown in webhook post")
	flag.BoolVar(&dryRun, "dry-run", false, "print what would be sent without posting")
	flag.Parse()

	if err := run(targetArg, message, senderName, dryRun); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(targetArg, message, senderName string, dryRun bool) error {
	if strings.TrimSpace(targetArg) == "" {
		return errors.New("--target is required")
	}
	if strings.TrimSpace(message) == "" {
		return errors.New("--message is required")
	}

	t, err := parseTarget(targetArg)
	if err != nil {
		return err
	}

	webhookURL := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL"))
	if webhookURL == "" {
		return errors.New("DISCORD_WEBHOOK_URL is not set")
	}

	payload := webhookPayload{Content: message}
	if strings.TrimSpace(senderName) != "" {
		payload.Username = senderName
	}

	resolvedURL, err := buildWebhookURL(webhookURL, t)
	if err != nil {
		return err
	}
	if t.kind == "discord-channel" {
		if err := verifyWebhookChannelMatch(webhookURL, t.id); err != nil {
			return err
		}
	}

	if dryRun {
		out := successResult{Target: targetArg, DryRun: true}
		return json.NewEncoder(os.Stdout).Encode(out)
	}

	res, err := postWebhook(resolvedURL, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 16*1024))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("discord webhook returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	result := successResult{
		Target: targetArg,
		DryRun: false,
	}

	if len(body) == 0 {
		return errors.New("discord webhook returned empty body despite wait=true")
	}

	var parsed struct {
		ID        string `json:"id"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("decode webhook response: %w", err)
	}
	if parsed.ID == "" || parsed.Timestamp == "" {
		return errors.New("discord webhook response missing id or timestamp")
	}
	result.MessageID = parsed.ID
	result.Timestamp = parsed.Timestamp

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}

func parseTarget(v string) (target, error) {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return target{}, fmt.Errorf("invalid target %q: expected discord-channel:<id> or discord-thread:<id>", v)
	}
	kind := strings.TrimSpace(parts[0])
	id := strings.TrimSpace(parts[1])
	if id == "" {
		return target{}, fmt.Errorf("invalid target %q: missing id", v)
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return target{}, fmt.Errorf("invalid target %q: id must be numeric", v)
		}
	}
	switch kind {
	case "discord-channel", "discord-thread":
		return target{kind: kind, id: id}, nil
	default:
		return target{}, fmt.Errorf("invalid target kind %q", kind)
	}
}

func buildWebhookURL(raw string, t target) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid DISCORD_WEBHOOK_URL: %w", err)
	}
	q := u.Query()
	q.Set("wait", "true")
	if t.kind == "discord-thread" {
		q.Set("thread_id", t.id)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func verifyWebhookChannelMatch(rawWebhookURL, expectedChannelID string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawWebhookURL, nil)
	if err != nil {
		return fmt.Errorf("build webhook verify request: %w", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verify webhook channel: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 16*1024))
	if err != nil {
		return fmt.Errorf("read webhook metadata: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("verify webhook channel failed: %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var meta struct {
		ChannelID string `json:"channel_id"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return fmt.Errorf("decode webhook metadata: %w", err)
	}
	if meta.ChannelID == "" {
		return errors.New("webhook metadata missing channel_id")
	}
	if meta.ChannelID != expectedChannelID {
		return fmt.Errorf("target channel mismatch: expected=%s actual=%s", expectedChannelID, meta.ChannelID)
	}
	return nil
}

func postWebhook(webhookURL string, payload webhookPayload) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post webhook: %w", err)
	}
	return res, nil
}
