package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MightyToolkit/mightymonitor-agent/internal/buffer"
	"github.com/MightyToolkit/mightymonitor-agent/internal/client"
	"github.com/MightyToolkit/mightymonitor-agent/internal/config"
	"github.com/MightyToolkit/mightymonitor-agent/internal/metrics"
)

const (
	defaultConfigPath = "/etc/mightymonitor/config.json"
	defaultBufferPath = "/var/lib/mightymonitor/buffer.jsonl"
	legacyBufferPath  = "/var/lib/mightymonitor/pending-payloads.jsonl"
	defaultBufferSize = 10
)

var Version = "0.1.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "print-payload":
			if err := runPrintPayload(); err != nil {
				log.Fatalf("print-payload failed: %v", err)
			}
			return
		case "enroll":
			if err := runEnroll(os.Args[2:]); err != nil {
				log.Fatalf("%v", err)
			}
			return
		case "send":
			if err := runSend(); err != nil {
				log.Fatalf("%v", err)
			}
			return
		case "version", "--version", "-v":
			fmt.Println(Version)
			return
		}
	}

	fmt.Printf("mightymonitor-agent %s\n", Version)
}

func runPrintPayload() error {
	hostID := "(not enrolled)"
	cfg, err := config.Load(defaultConfigPath)
	if err == nil {
		if cfg.HostID != "" {
			hostID = cfg.HostID
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	payload, err := buildPayload(hostID)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func runEnroll(args []string) error {
	fs := flag.NewFlagSet("enroll", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	token := fs.String("token", "", "enrollment token")
	server := fs.String("server", "", "server URL")
	allowInsecure := fs.Bool("allow-insecure", false, "allow http://localhost only")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*token) == "" {
		return errors.New("Error: --token is required")
	}
	if strings.TrimSpace(*server) == "" {
		return errors.New("Error: --server is required")
	}

	hostname := metrics.GetHostname()
	cfg := &config.Config{ServerURL: *server}
	cli := client.NewClientWithOptions(cfg, *allowInsecure)
	resp, err := cli.Enroll(context.Background(), *token, hostname)
	if err != nil {
		var httpErr *client.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.StatusCode == http.StatusGone {
				return errors.New("Error: enrollment token has expired. Please generate a new one from the Mighty Monitor app.")
			}
			if isEnrollmentTokenUsedError(httpErr) {
				return errors.New("Error: enrollment token has already been used.")
			}
		}
		return fmt.Errorf("Error: enrollment failed: %w", err)
	}

	cfg.HostID = resp.HostID
	cfg.HostToken = resp.HostToken
	cfg.AllowInsecureLocalhost = *allowInsecure
	if err := config.Save(defaultConfigPath, cfg); err != nil {
		return fmt.Errorf("Error: failed to write config: %w", err)
	}

	fmt.Printf("Enrolled successfully. Host ID: %s\n", resp.HostID)
	return nil
}

func runSend() error {
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		log.Printf("Warning: failed to load config: %v", err)
		return nil
	}

	payload, err := buildPayload(cfg.HostID)
	if err != nil {
		log.Printf("Warning: failed to collect metrics: %v", err)
		return nil
	}

	migrateLegacyBufferFile(defaultBufferPath, legacyBufferPath)

	payloadBuffer := buffer.NewBuffer(defaultBufferPath, defaultBufferSize)
	cli := client.NewClientWithOptions(cfg, cfg.AllowInsecureLocalhost)

	bufferedCount, err := payloadBuffer.Count()
	if err != nil {
		log.Printf("Warning: failed to inspect buffer: %v", err)
		bufferedCount = 0
	}

	if bufferedCount > 0 {
		pending, err := payloadBuffer.Flush()
		if err != nil {
			log.Printf("Warning: failed to read buffer: %v", err)
			pending = nil
		}

		if len(pending) > 0 {
			batchResp, err := cli.SendBatch(context.Background(), pending)
			if err != nil {
				if pushErr := payloadBuffer.Push(payload); pushErr != nil {
					log.Printf("Warning: batch send failed and current payload buffering failed: send_err=%v buffer_err=%v", err, pushErr)
				} else {
					log.Printf("Warning: batch send failed; current payload buffered: %v", err)
				}
				return nil
			}
			if batchResp.Rejected > 0 {
				if pushErr := payloadBuffer.Push(payload); pushErr != nil {
					log.Printf("Warning: batch send partially rejected (%d rejected) and buffering current payload failed: %v", batchResp.Rejected, pushErr)
				} else {
					log.Printf("Warning: batch send partially rejected (%d rejected); keeping buffered payloads and buffering current payload", batchResp.Rejected)
				}
				return nil
			}

			if err := payloadBuffer.Clear(); err != nil {
				log.Printf("Warning: flushed buffered payloads but failed to clear buffer: %v", err)
			}
		}
	}

	response, err := cli.SendPayload(context.Background(), payload)
	if err != nil {
		if pushErr := payloadBuffer.Push(payload); pushErr != nil {
			log.Printf("Warning: send failed and buffering failed: send_err=%v buffer_err=%v", err, pushErr)
		} else {
			log.Printf("Warning: send failed; payload buffered: %v", err)
		}
		return nil
	}

	if response.ClockSkew {
		log.Printf("Warning: server detected clock skew > 5 minutes. Consider running ntpd/chrony.")
	}
	return nil
}

func buildPayload(hostID string) (*metrics.Payload, error) {
	payload, err := metrics.Collect()
	if err != nil {
		return nil, err
	}

	payload.HostID = hostID
	payload.AgentVersion = Version
	payload.TS = time.Now().Unix()
	return payload, nil
}

func isEnrollmentTokenUsedError(httpErr *client.HTTPError) bool {
	if httpErr == nil || httpErr.StatusCode != http.StatusBadRequest {
		return false
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(httpErr.Body), &body); err == nil {
		return strings.EqualFold(strings.TrimSpace(body.Error), "enrollment token already used")
	}
	return strings.EqualFold(strings.TrimSpace(httpErr.Body), "enrollment token already used")
}

func migrateLegacyBufferFile(currentPath string, oldPath string) {
	if currentPath == oldPath {
		return
	}

	if _, err := os.Stat(currentPath); err == nil {
		return
	}
	if _, err := os.Stat(oldPath); err != nil {
		return
	}
	if err := os.Rename(oldPath, currentPath); err != nil {
		log.Printf("Warning: failed to migrate legacy buffer file from %s to %s: %v", oldPath, currentPath, err)
	}
}
