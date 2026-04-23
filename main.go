package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/joho/godotenv"
)

type config struct {
	Pattern compiledPattern
	Workers int
}

type result struct {
	Address    string
	PublicKey  string
	PrivateKey string
}

type compiledPattern struct {
	Indexes  []int
	Expected []byte
}

func main() {
	_ = godotenv.Load()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	match, err := findMatch(cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("address=%s\n", match.Address)
	fmt.Printf("public_key=%s\n", match.PublicKey)
	fmt.Printf("private_key=%s\n", match.PrivateKey)
}

func loadConfig() (config, error) {
	rawPattern := normalizePattern(os.Getenv("ADDRESS_PATTERN"))
	if rawPattern == "" {
		return config{}, fmt.Errorf("ADDRESS_PATTERN is required")
	}

	if len(rawPattern) != 40 {
		return config{}, fmt.Errorf("ADDRESS_PATTERN must be exactly 40 hex characters after 0x")
	}

	for _, ch := range rawPattern {
		if ch == '?' {
			continue
		}

		if !strings.ContainsRune("0123456789abcdef", ch) {
			return config{}, fmt.Errorf("ADDRESS_PATTERN only supports lowercase hex characters and '?' wildcards")
		}
	}

	workers := runtime.NumCPU()
	if rawWorkers := strings.TrimSpace(os.Getenv("WORKERS")); rawWorkers != "" {
		parsedWorkers, err := parseWorkerCount(rawWorkers)
		if err != nil {
			return config{}, fmt.Errorf("WORKERS must be zero or a positive integer: %w", err)
		}

		if parsedWorkers > 0 {
			workers = parsedWorkers
		}
	}

	pattern := compilePattern(rawPattern)

	return config{
		Pattern: pattern,
		Workers: workers,
	}, nil
}

func normalizePattern(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	return strings.TrimPrefix(trimmed, "0x")
}

func parseWorkerCount(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	if parsed < 0 {
		return 0, fmt.Errorf("value must be at least zero")
	}

	return parsed, nil
}

func findMatch(cfg config) (result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan result, 1)
	errs := make(chan error, cfg.Workers)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if err := worker(ctx, cfg.Pattern, results); err != nil {
				select {
				case errs <- err:
				default:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	select {
	case match := <-results:
		cancel()
		return match, nil
	case err := <-errs:
		if err == nil {
			return result{}, fmt.Errorf("generator stopped unexpectedly")
		}

		cancel()
		return result{}, err
	}
}

func worker(ctx context.Context, pattern compiledPattern, results chan<- result) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		privateKey, err := crypto.GenerateKey()
		if err != nil {
			return err
		}

		address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
		if !pattern.Matches(address) {
			continue
		}

		match := result{
			Address:    address,
			PublicKey:  encodePublicKey(&privateKey.PublicKey),
			PrivateKey: encodePrivateKey(privateKey),
		}

		select {
		case results <- match:
			return nil
		case <-ctx.Done():
			return nil
		}
	}
}

func compilePattern(raw string) compiledPattern {
	indexes := make([]int, 0, len(raw))
	expected := make([]byte, 0, len(raw))

	for i := 0; i < len(raw); i++ {
		if raw[i] == '?' {
			continue
		}

		indexes = append(indexes, i)
		expected = append(expected, raw[i])
	}

	return compiledPattern{
		Indexes:  indexes,
		Expected: expected,
	}
}

func (p compiledPattern) Matches(address string) bool {
	for i, idx := range p.Indexes {
		if lowerHex(address[idx+2]) != p.Expected[i] {
			return false
		}
	}

	return true
}

func lowerHex(ch byte) byte {
	if ch >= 'A' && ch <= 'F' {
		return ch + ('a' - 'A')
	}

	return ch
}

func encodePublicKey(publicKey *ecdsa.PublicKey) string {
	return hex.EncodeToString(crypto.FromECDSAPub(publicKey))
}

func encodePrivateKey(privateKey *ecdsa.PrivateKey) string {
	return hex.EncodeToString(crypto.FromECDSA(privateKey))
}
