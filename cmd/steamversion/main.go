package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const steamInfoURL = "https://raw.githubusercontent.com/SteamDatabase/GameTracking-CS2/master/csgo/steam.inf"

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}

	versionMu      sync.RWMutex
	currentVersion string
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	version, err := fetchSteamVersion()
	if err != nil {
		log.Fatalf("failed to fetch Steam version: %v", err)
	}
	setGameVersion(version)

	go autoUpdateSteamVersion(ctx)

	<-ctx.Done()
	log.Println("shutting down")
}

func fetchSteamVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, steamInfoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request %s: %w", steamInfoURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s from %s", resp.Status, steamInfoURL)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PatchVersion=") {
			version := strings.TrimSpace(strings.TrimPrefix(line, "PatchVersion="))
			if version == "" {
				return "", errors.New("patch version is empty")
			}
			return version, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return "", errors.New("patch version not found in steam.inf")
}

func autoUpdateSteamVersion(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			version, err := fetchSteamVersion()
			if err != nil {
				log.Printf("failed to refresh Steam version: %v", err)
				continue
			}

			if version != getGameVersion() {
				setGameVersion(version)
			}
		}
	}
}

func getGameVersion() string {
	versionMu.RLock()
	defer versionMu.RUnlock()
	return currentVersion
}

func setGameVersion(version string) {
	versionMu.Lock()
	defer versionMu.Unlock()

	if currentVersion == version {
		return
	}

	if currentVersion == "" {
		log.Printf("Steam version set to %s", version)
	} else {
		log.Printf("Steam version updated from %s to %s", currentVersion, version)
	}

	currentVersion = version
}
