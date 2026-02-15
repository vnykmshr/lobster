// Package main provides the command-line interface for the Lobster load testing tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/1mb-dev/lobster/v2/internal/cli"
	"github.com/1mb-dev/lobster/v2/internal/domain"
	"github.com/1mb-dev/lobster/v2/internal/reporter"
	"github.com/1mb-dev/lobster/v2/internal/tester"
	"github.com/1mb-dev/lobster/v2/internal/util"
	"github.com/1mb-dev/lobster/v2/internal/validator"
)

// version is set at build time via ldflags: -X main.version=X.Y.Z
// Default fallback if not set during build
var version = "dev"

func main() {
	var (
		configPath         = flag.String("config", "", "Path to configuration file (JSON)")
		baseURL            = flag.String("url", "", "Base URL to test")
		concurrency        = flag.Int("concurrency", 0, "Number of concurrent workers")
		duration           = flag.String("duration", "", "Test duration (e.g., 30s, 5m, 1h)")
		timeout            = flag.String("timeout", "", "Request timeout")
		rate               = flag.Float64("rate", 0, "Requests per second limit")
		userAgent          = flag.String("user-agent", "", "User agent string")
		followLinks        = flag.Bool("follow-links", true, "Follow links found in pages")
		maxDepth           = flag.Int("max-depth", 0, "Maximum crawl depth")
		queueSize          = flag.Int("queue-size", 0, "URL queue buffer size (default: 10000)")
		respect429         = flag.Bool("respect-429", true, "Respect HTTP 429 with exponential backoff")
		dryRun             = flag.Bool("dry-run", false, "Discover URLs without making test requests")
		insecureSkipVerify = flag.Bool("insecure-skip-verify", false, "INSECURE: Skip TLS certificate verification")
		allowPrivateIPs    = flag.Bool("allow-private-ips", false, "Allow private/localhost IPs (for internal testing)")
		ignoreRobots       = flag.Bool("ignore-robots", false, "Ignore robots.txt directives (use responsibly)")
		outputFile         = flag.String("output", "", "Output file for results (JSON)")
		verbose            = flag.Bool("verbose", false, "Verbose logging")
		noProgress         = flag.Bool("no-progress", false, "Disable progress updates")
		showVersion        = flag.Bool("version", false, "Show version information")
		showHelp           = flag.Bool("help", false, "Show help message")
		compareAgainst     = flag.String("compare", "", "Compare against target (e.g., 'Ghost', 'WordPress')")

		// Authentication flags
		authType          = flag.String("auth-type", "", "Authentication type: basic, bearer, cookie, header")
		authUsername      = flag.String("auth-username", "", "Username for basic authentication")
		authPasswordStdin = flag.Bool("auth-password-stdin", false, "Read password from stdin (one line)")
		authTokenStdin    = flag.Bool("auth-token-stdin", false, "Read bearer token from stdin (one line)")
		authHeader        = flag.String("auth-header", "", "Custom header (Name:Value)")
	)
	flag.Parse()

	// Setup logger early so all errors use consistent output format
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	if *showVersion {
		fmt.Printf("Lobster v%s\n", version)
		return
	}

	if *showHelp {
		cli.ShowHelpMessage(version)
		return
	}

	// Load configuration
	cfg, err := cli.LoadConfiguration(*configPath, &cli.ConfigOptions{
		BaseURL:            *baseURL,
		Concurrency:        *concurrency,
		Duration:           *duration,
		Timeout:            *timeout,
		Rate:               *rate,
		UserAgent:          *userAgent,
		FollowLinks:        *followLinks,
		MaxDepth:           *maxDepth,
		QueueSize:          *queueSize,
		Respect429:         *respect429,
		DryRun:             *dryRun,
		InsecureSkipVerify: *insecureSkipVerify,
		IgnoreRobots:       *ignoreRobots,
		OutputFile:         *outputFile,
		Verbose:            *verbose,
		AuthType:           *authType,
		AuthUsername:       *authUsername,
		AuthPasswordStdin:  *authPasswordStdin,
		AuthTokenStdin:     *authTokenStdin,
		AuthHeader:         *authHeader,
	})
	if err != nil {
		logger.Error("Configuration error",
			"error", err,
			"hint", "Check your config file syntax or command-line flags")
		os.Exit(1)
	}

	// Validate and enforce rate limit safety
	if validateErr := cli.ValidateRateLimit(&cfg.Rate); validateErr != nil {
		logger.Error("Invalid rate limit",
			"error", validateErr,
			"hint", "Use -rate flag with a value >= 0.1")
		os.Exit(1)
	}

	// Validate base URL (SSRF protection)
	if validateErr := util.ValidateBaseURL(cfg.BaseURL, *allowPrivateIPs); validateErr != nil {
		logger.Error("Invalid base URL", "error", validateErr)
		os.Exit(1)
	}

	// Warn about allowing private IPs
	if *allowPrivateIPs {
		cli.PrintWarningBox("SECURITY WARNING", []string{
			"Private/localhost IPs are ALLOWED (-allow-private-ips)",
			"",
			"This bypasses SSRF protection. Only use for:",
			"  • Testing internal/local services you own",
			"  • Development environments",
			"",
			"NEVER use this with untrusted URL inputs!",
		})
	}

	// Require explicit env var confirmation for insecure TLS
	if cfg.InsecureSkipVerify {
		if os.Getenv("LOBSTER_INSECURE_TLS") != "true" {
			logger.Error("Insecure TLS requires explicit confirmation",
				"required_env", "LOBSTER_INSECURE_TLS=true",
				"reason", "security safeguard to prevent accidental TLS bypass",
				"implications", []string{
					"man-in-the-middle attacks become possible",
					"certificate validation is completely bypassed",
					"use only for testing with self-signed certificates",
				})
			os.Exit(1)
		}

		// Show warning after env var check passes
		cli.PrintWarningBox("SECURITY WARNING", []string{
			"TLS certificate verification is DISABLED (-insecure-skip-verify)",
			"",
			"This makes you vulnerable to man-in-the-middle attacks!",
			"",
			"Only use this for:",
			"  • Testing with self-signed certificates",
			"  • Internal development environments",
			"",
			"NEVER use this in production or with untrusted networks!",
		})
	}

	// Warn about ignoring robots.txt
	if cfg.IgnoreRobots {
		cli.PrintWarningBox("ETHICAL WARNING", []string{
			"Ignoring robots.txt directives (-ignore-robots)",
			"",
			"You are bypassing the website owner's crawling preferences!",
			"",
			"Only use this when:",
			"  • You OWN the website being tested",
			"  • You have explicit written permission",
			"",
			"Unauthorized testing may violate terms of service or laws!",
		})
	}

	// Parse duration
	testDuration, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		logger.Error("Invalid duration format",
			"error", err,
			"hint", "Use format like: 30s, 5m, 1h (e.g., -duration 2m)")
		os.Exit(1)
	}

	// Parse timeout
	requestTimeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		logger.Error("Invalid timeout format",
			"error", err,
			"hint", "Use format like: 30s, 5m, 1h (e.g., -timeout 30s)")
		os.Exit(1)
	}

	// Initialize stress tester config
	testerConfig := domain.TesterConfig{
		BaseURL:            cfg.BaseURL,
		Concurrency:        cfg.Concurrency,
		RequestTimeout:     requestTimeout,
		UserAgent:          cfg.UserAgent,
		Auth:               cfg.Auth,
		FollowLinks:        cfg.FollowLinks,
		MaxDepth:           cfg.MaxDepth,
		QueueSize:          cfg.QueueSize,
		Respect429:         cfg.Respect429,
		DryRun:             cfg.DryRun,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		IgnoreRobots:       cfg.IgnoreRobots,
		Rate:               cfg.Rate,
		Verbose:            cfg.Verbose,
		NoProgress:         *noProgress,
	}

	// Run stress test in a function that handles its own context
	results, err := runStressTest(testerConfig, testDuration, logger)
	if err != nil {
		logger.Error("Stress test failed", "error", err)
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
			logger.Error("Cannot write results",
				"file", cfg.OutputFile,
				"error", err,
				"hint", "Check file permissions and disk space")
			os.Exit(1)
		}
		logger.Info("Results saved", "file", cfg.OutputFile)

		// Generate HTML report
		htmlFile := strings.TrimSuffix(cfg.OutputFile, filepath.Ext(cfg.OutputFile)) + ".html"
		err = rep.GenerateHTML(htmlFile)
		if err != nil {
			logger.Error("Cannot write HTML report", "file", htmlFile, "error", err)
		} else {
			logger.Info("HTML report generated", "file", htmlFile)
		}
	}
}

// runStressTest executes the stress test with proper context management.
// This function encapsulates context creation and cancellation to ensure
// deferred cleanup runs correctly regardless of how the function exits.
func runStressTest(config domain.TesterConfig, duration time.Duration, logger *slog.Logger) (*domain.TestResults, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	stressTester, err := tester.New(config, logger)
	if err != nil {
		return nil, fmt.Errorf("tester initialization failed: %w", err)
	}

	logger.Info("Starting stress test",
		"base_url", config.BaseURL,
		"concurrency", config.Concurrency,
		"rate", config.Rate,
		"follow_links", config.FollowLinks,
		"max_depth", config.MaxDepth)

	return stressTester.Run(ctx)
}
