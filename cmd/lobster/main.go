// Package main provides the command-line interface for the Lobster load testing tool.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vnykmshr/lobster/internal/config"
	"github.com/vnykmshr/lobster/internal/domain"
	"github.com/vnykmshr/lobster/internal/reporter"
	"github.com/vnykmshr/lobster/internal/tester"
	"github.com/vnykmshr/lobster/internal/validator"
)

const version = "0.1.0"

func main() {
	var (
		configPath        = flag.String("config", "", "Path to configuration file (JSON)")
		baseURL           = flag.String("url", "", "Base URL to test")
		concurrency       = flag.Int("concurrency", 0, "Number of concurrent workers")
		duration          = flag.String("duration", "", "Test duration (e.g., 30s, 5m, 1h)")
		timeout           = flag.String("timeout", "", "Request timeout")
		rate              = flag.Float64("rate", 0, "Requests per second limit")
		userAgent         = flag.String("user-agent", "", "User agent string")
		followLinks       = flag.Bool("follow-links", true, "Follow links found in pages")
		maxDepth          = flag.Int("max-depth", 0, "Maximum crawl depth")
		queueSize         = flag.Int("queue-size", 0, "URL queue buffer size (default: 10000)")
		outputFile        = flag.String("output", "", "Output file for results (JSON)")
		verbose           = flag.Bool("verbose", false, "Verbose logging")
		showVersion       = flag.Bool("version", false, "Show version information")
		showHelp          = flag.Bool("help", false, "Show help message")
		compareAgainst    = flag.String("compare", "", "Compare against target (e.g., 'Ghost', 'WordPress')")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Lobster v%s\n", version)
		return
	}

	if *showHelp {
		showHelpMessage()
		return
	}

	// Load configuration
	cfg, err := loadConfiguration(*configPath, &configOptions{
		baseURL:     *baseURL,
		concurrency: *concurrency,
		duration:    *duration,
		timeout:     *timeout,
		rate:        *rate,
		userAgent:   *userAgent,
		followLinks: *followLinks,
		maxDepth:    *maxDepth,
		queueSize:   *queueSize,
		outputFile:  *outputFile,
		verbose:     *verbose,
	})
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate and enforce rate limit safety
	if err := validateRateLimit(&cfg.Rate); err != nil {
		log.Fatalf("Rate limit validation failed: %v", err)
	}

	// Setup logger
	logLevel := slog.LevelInfo
	if cfg.Verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Parse duration
	testDuration, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		log.Fatalf("Invalid duration: %v", err)
	}

	// Parse timeout
	requestTimeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		log.Fatalf("Invalid timeout: %v", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Initialize stress tester
	testerConfig := domain.TesterConfig{
		BaseURL:        cfg.BaseURL,
		Concurrency:    cfg.Concurrency,
		RequestTimeout: requestTimeout,
		UserAgent:      cfg.UserAgent,
		FollowLinks:    cfg.FollowLinks,
		MaxDepth:       cfg.MaxDepth,
		QueueSize:      cfg.QueueSize,
		Rate:           cfg.Rate,
	}

	stressTester, err := tester.New(testerConfig, logger)
	if err != nil {
		cancel()
		log.Fatalf("Failed to create tester: %v", err) //nolint:gocritic // cancel() is called explicitly before exit
	}

	// Run stress test
	logger.Info("Starting stress test",
		"base_url", cfg.BaseURL,
		"concurrency", cfg.Concurrency,
		"duration", cfg.Duration,
		"rate", cfg.Rate,
		"follow_links", cfg.FollowLinks,
		"max_depth", cfg.MaxDepth)

	results, err := stressTester.Run(ctx)
	if err != nil {
		log.Printf("Stress test failed: %v", err)
		os.Exit(1)
	}

	// Create validator
	var performanceValidator *validator.Validator
	if *compareAgainst != "" {
		performanceValidator = validator.NewWithComparison(cfg.PerformanceTargets, *compareAgainst)
	} else {
		performanceValidator = validator.New(cfg.PerformanceTargets)
	}
	performanceValidator.ValidateResults(results)

	// Create reporter
	rep := reporter.New(results)

	// Print console summary
	rep.PrintSummary()

	// Print performance validation
	performanceValidator.PrintValidationReport()

	// Output results to file
	if cfg.OutputFile != "" {
		// Add validation data to results
		results.PerformanceValidation = performanceValidator.GetValidationSummary()

		// Save JSON results
		err = rep.GenerateJSON(cfg.OutputFile)
		if err != nil {
			cancel()
			log.Fatalf("Failed to save results: %v", err)
		}
		logger.Info("Results saved", "file", cfg.OutputFile)

		// Generate HTML report
		htmlFile := strings.TrimSuffix(cfg.OutputFile, filepath.Ext(cfg.OutputFile)) + ".html"
		err = rep.GenerateHTML(htmlFile)
		if err != nil {
			logger.Error("Failed to generate HTML report", "error", err)
		} else {
			logger.Info("HTML report generated", "file", htmlFile)
		}
	}
}

type configOptions struct {
	baseURL     string
	duration    string
	timeout     string
	userAgent   string
	outputFile  string
	rate        float64
	concurrency int
	maxDepth    int
	queueSize   int
	followLinks bool
	verbose     bool
}

func loadConfiguration(configPath string, opts *configOptions) (*domain.Config, error) {
	loader := config.NewLoader()

	var cfg *domain.Config

	if configPath != "" {
		// Load from file
		loadedCfg, err := loader.LoadFromFile(configPath)
		if err != nil {
			return nil, err
		}
		cfg = loadedCfg
	} else {
		// Start with defaults
		defaultCfg := domain.DefaultConfig()
		cfg = &defaultCfg
	}

	// Override with CLI flags (if provided)
	if opts.baseURL != "" {
		cfg.BaseURL = opts.baseURL
	}
	if opts.concurrency != 0 {
		cfg.Concurrency = opts.concurrency
	}
	if opts.duration != "" {
		cfg.Duration = opts.duration
	}
	if opts.timeout != "" {
		cfg.Timeout = opts.timeout
	}
	if opts.rate != 0 {
		cfg.Rate = opts.rate
	}
	if opts.userAgent != "" {
		cfg.UserAgent = opts.userAgent
	}
	if opts.maxDepth != 0 {
		cfg.MaxDepth = opts.maxDepth
	}
	if opts.queueSize != 0 {
		cfg.QueueSize = opts.queueSize
	}
	if opts.outputFile != "" {
		cfg.OutputFile = opts.outputFile
	}
	cfg.FollowLinks = opts.followLinks
	cfg.Verbose = opts.verbose

	// Merge with defaults for any missing values
	cfg = loader.MergeWithDefaults(cfg)

	return cfg, nil
}

func showHelpMessage() {
	fmt.Println(`Lobster - Intelligent Web Stress Testing Tool

USAGE:
    lobster [OPTIONS]

OPTIONS:
    -config string
        Path to configuration file (JSON format)
    -url string
        Base URL to test (default: http://localhost:3000)
    -concurrency int
        Number of concurrent workers (default: 5)
    -duration string
        Test duration (e.g., 30s, 5m, 1h) (default: 2m)
    -timeout string
        Request timeout (default: 30s)
    -rate float
        Requests per second limit (default: 2.0)
        Safety: Minimum 0.1 req/s enforced
        Warning prompt for rates < 1.0 req/s
    -user-agent string
        User agent string (default: Lobster/1.0)
    -follow-links
        Follow links found in pages (default: true)
    -max-depth int
        Maximum crawl depth (default: 3)
    -queue-size int
        URL queue buffer size (default: 10000)
        Memory usage: ~8 bytes per queue slot
    -output string
        Output file for results (JSON format)
    -verbose
        Enable verbose logging
    -compare string
        Compare performance against target (e.g., Ghost, WordPress)
    -version
        Show version information
    -help
        Show this help message

EXAMPLES:
    # Basic stress test
    lobster -url http://localhost:3000

    # High concurrency test with custom duration
    lobster -url http://localhost:3000 -concurrency 50 -duration 5m

    # Test with rate limiting and output
    lobster -url http://localhost:3000 -rate 10 -output results.json

    # Use configuration file
    lobster -config myconfig.json

    # Compare against competitor
    lobster -url http://localhost:3000 -compare "Ghost"

    # Quick validation test
    lobster -url http://localhost:3000 -duration 30s -concurrency 5

CONFIGURATION FILE EXAMPLE:
    {
      "base_url": "http://localhost:3000",
      "concurrency": 10,
      "duration": "5m",
      "timeout": "30s",
      "rate": 10.0,
      "user_agent": "Lobster/1.0",
      "follow_links": true,
      "max_depth": 3,
      "queue_size": 10000,
      "output_file": "results.json",
      "verbose": false,
      "performance_targets": {
        "requests_per_second": 100,
        "avg_response_time_ms": 50,
        "p95_response_time_ms": 100,
        "p99_response_time_ms": 200,
        "success_rate": 99.0,
        "error_rate": 1.0
      }
    }

DOCUMENTATION:
    Full documentation: https://github.com/vnykmshr/lobster
    Report issues: https://github.com/vnykmshr/lobster/issues

VERSION:
    Lobster v` + version + `

Made with ❤️  for developers who value simplicity and power`)
}

// validateRateLimit enforces safe rate limiting to prevent accidental DoS
func validateRateLimit(rate *float64) error {
	const (
		minRate  = 0.1  // Minimum allowed rate (requests per second)
		warnRate = 1.0  // Warning threshold for low rates
	)

	// Rate of 0 means no rate limiting (unlimited)
	if *rate == 0 {
		fmt.Fprintf(os.Stderr, "\n⚠️  WARNING: No rate limiting enabled (rate=0)\n")
		fmt.Fprintf(os.Stderr, "   This will send requests as fast as possible.\n")
		fmt.Fprintf(os.Stderr, "   Make sure you have permission to test the target server.\n\n")
		return nil
	}

	// Enforce minimum rate to prevent extremely slow tests
	if *rate > 0 && *rate < minRate {
		fmt.Fprintf(os.Stderr, "\n⚠️  WARNING: Rate %.2f req/s is below minimum %.2f req/s\n", *rate, minRate)
		fmt.Fprintf(os.Stderr, "   Adjusting to minimum rate of %.2f req/s\n", minRate)
		fmt.Fprintf(os.Stderr, "   Rationale: Extremely low rates may indicate configuration error.\n\n")
		*rate = minRate
		return nil
	}

	// Warn about very low rates and prompt for confirmation if interactive
	if *rate < warnRate {
		fmt.Fprintf(os.Stderr, "\n⚠️  WARNING: Low rate limit detected (%.2f req/s)\n", *rate)
		fmt.Fprintf(os.Stderr, "   This will send requests very slowly:\n")
		fmt.Fprintf(os.Stderr, "   - %.2f requests per second\n", *rate)
		fmt.Fprintf(os.Stderr, "   - ~%.0f requests per minute\n", *rate*60)
		fmt.Fprintf(os.Stderr, "   - Test may take a long time to complete\n\n")

		// If interactive terminal, prompt for confirmation
		if isInteractiveTerminal() {
			fmt.Fprintf(os.Stderr, "   Do you want to continue with this rate? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading confirmation: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("test cancelled by user")
			}
			fmt.Fprintf(os.Stderr, "\n")
		} else {
			fmt.Fprintf(os.Stderr, "   Continuing in non-interactive mode...\n\n")
		}
	}

	return nil
}

// isInteractiveTerminal checks if the program is running in an interactive terminal
func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a character device (terminal) rather than a pipe or file
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
