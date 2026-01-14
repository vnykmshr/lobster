// Package main provides the command-line interface for the Lobster load testing tool.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"github.com/vnykmshr/lobster/internal/util"
	"github.com/vnykmshr/lobster/internal/validator"
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
		baseURL:            *baseURL,
		concurrency:        *concurrency,
		duration:           *duration,
		timeout:            *timeout,
		rate:               *rate,
		userAgent:          *userAgent,
		followLinks:        *followLinks,
		maxDepth:           *maxDepth,
		queueSize:          *queueSize,
		respect429:         *respect429,
		dryRun:             *dryRun,
		insecureSkipVerify: *insecureSkipVerify,
		ignoreRobots:       *ignoreRobots,
		outputFile:         *outputFile,
		verbose:            *verbose,
		authType:           *authType,
		authUsername:       *authUsername,
		authPasswordStdin:  *authPasswordStdin,
		authTokenStdin:     *authTokenStdin,
		authHeader:         *authHeader,
	})
	if err != nil {
		log.Fatalf("Configuration error: %v\nCheck your config file syntax or command-line flags", err)
	}

	// Validate and enforce rate limit safety
	if validateErr := validateRateLimit(&cfg.Rate); validateErr != nil {
		log.Fatalf("Invalid rate limit: %v\nUse -rate flag with a value >= 0.1", validateErr)
	}

	// Validate base URL (SSRF protection)
	if validateErr := util.ValidateBaseURL(cfg.BaseURL, *allowPrivateIPs); validateErr != nil {
		log.Fatalf("Invalid base URL: %v", validateErr)
	}

	// Warn about allowing private IPs
	if *allowPrivateIPs {
		printWarningBox("SECURITY WARNING", []string{
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
			log.Fatalf(`Insecure TLS verification disabled but LOBSTER_INSECURE_TLS=true not set.

This is a security safeguard to prevent accidental TLS bypass in production.

To proceed, set the environment variable:
  export LOBSTER_INSECURE_TLS=true

Only do this if you understand the security implications:
  - Man-in-the-middle attacks become possible
  - Certificate validation is completely bypassed
  - Use only for testing with self-signed certificates`)
		}

		// Show warning after env var check passes
		printWarningBox("SECURITY WARNING", []string{
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
		printWarningBox("ETHICAL WARNING", []string{
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
		log.Fatalf("Invalid duration format: %v\nUse format like: 30s, 5m, 1h (e.g., -duration 2m)", err)
	}

	// Parse timeout
	requestTimeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		log.Fatalf("Invalid timeout format: %v\nUse format like: 30s, 5m, 1h (e.g., -timeout 30s)", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), testDuration)
	defer cancel()

	// Initialize stress tester
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

	stressTester, err := tester.New(testerConfig, logger)
	if err != nil {
		cancel()
		log.Fatalf("Tester initialization failed: %v\nCheck your configuration and base URL", err) //nolint:gocritic // cancel() is called explicitly before exit
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
			log.Fatalf("Cannot write results to %s: %v\nCheck file permissions and disk space", cfg.OutputFile, err)
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

type configOptions struct {
	baseURL            string
	duration           string
	timeout            string
	userAgent          string
	outputFile         string
	rate               float64
	concurrency        int
	maxDepth           int
	queueSize          int
	followLinks        bool
	respect429         bool
	dryRun             bool
	verbose            bool
	insecureSkipVerify bool
	ignoreRobots       bool
	authType           string
	authUsername       string
	authHeader         string
	authPasswordStdin  bool
	authTokenStdin     bool
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
	cfg.Respect429 = opts.respect429
	cfg.DryRun = opts.dryRun
	cfg.Verbose = opts.verbose
	cfg.InsecureSkipVerify = opts.insecureSkipVerify
	cfg.IgnoreRobots = opts.ignoreRobots

	// Build authentication configuration from CLI flags and environment variables
	authCfg, err := buildAuthConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("authentication configuration: %w", err)
	}
	if authCfg != nil {
		cfg.Auth = authCfg
	}

	// Merge with defaults for any missing values
	cfg = loader.MergeWithDefaults(cfg)

	return cfg, nil
}

// buildAuthConfig builds authentication configuration from environment variables and stdin.
// Credentials are read from:
// 1. Environment variables (LOBSTER_AUTH_PASSWORD, LOBSTER_AUTH_TOKEN, LOBSTER_AUTH_COOKIE)
// 2. Stdin when --auth-password-stdin or --auth-token-stdin flags are used
// CLI flags for credentials are intentionally not supported to prevent exposure in process lists.
func buildAuthConfig(opts *configOptions) (*domain.AuthConfig, error) {
	// Validate stdin flags are mutually exclusive (can only read one value from stdin)
	if opts.authPasswordStdin && opts.authTokenStdin {
		return nil, fmt.Errorf("--auth-password-stdin and --auth-token-stdin are mutually exclusive")
	}

	// Get credentials from environment variables
	password := os.Getenv("LOBSTER_AUTH_PASSWORD")
	token := os.Getenv("LOBSTER_AUTH_TOKEN")
	cookie := os.Getenv("LOBSTER_AUTH_COOKIE")

	// Read from stdin if requested (overrides env vars)
	if opts.authPasswordStdin {
		stdinPassword, err := readSecretFromStdin("password")
		if err != nil {
			return nil, err
		}
		password = stdinPassword
	}

	if opts.authTokenStdin {
		stdinToken, err := readSecretFromStdin("token")
		if err != nil {
			return nil, err
		}
		token = stdinToken
	}

	// Check if any auth configuration is provided
	hasAuth := opts.authType != "" || opts.authUsername != "" || opts.authHeader != "" ||
		password != "" || token != "" || cookie != ""

	if !hasAuth {
		return nil, nil
	}

	authCfg := &domain.AuthConfig{
		Type:     opts.authType,
		Username: opts.authUsername,
		Password: password,
		Token:    token,
	}

	// Parse cookie string (name=value) from env var
	if cookie != "" {
		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid LOBSTER_AUTH_COOKIE format: expected 'name=value', got %q", cookie)
		}
		authCfg.Cookies = make(map[string]string)
		authCfg.Cookies[parts[0]] = parts[1]
	}

	// Parse header string (Name:Value)
	if opts.authHeader != "" {
		parts := strings.SplitN(opts.authHeader, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid auth header format: expected 'Name:Value', got %q", opts.authHeader)
		}
		authCfg.Headers = make(map[string]string)
		authCfg.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return authCfg, nil
}

// readSecretFromStdin reads a single line from stdin for secure credential input.
func readSecretFromStdin(name string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading %s from stdin: %w", name, err)
	}
	return strings.TrimSpace(line), nil
}

// printWarningBox prints a formatted warning box to stderr.
// The box has a consistent width and styling for security/ethical warnings.
func printWarningBox(title string, lines []string) {
	const boxWidth = 68 // Inner content width

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "╔%s╗\n", strings.Repeat("═", boxWidth+2))
	fmt.Fprintf(os.Stderr, "║ %-*s ║\n", boxWidth, centerText(title, boxWidth))
	fmt.Fprintf(os.Stderr, "╠%s╣\n", strings.Repeat("═", boxWidth+2))

	for _, line := range lines {
		if line == "" {
			fmt.Fprintf(os.Stderr, "║ %-*s ║\n", boxWidth, "")
		} else {
			fmt.Fprintf(os.Stderr, "║  %-*s║\n", boxWidth-1, line)
		}
	}

	fmt.Fprintf(os.Stderr, "╚%s╝\n", strings.Repeat("═", boxWidth+2))
	fmt.Fprintf(os.Stderr, "\n")
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
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
    -respect-429
        Respect HTTP 429 with exponential backoff (default: true)
        Backoff: 1s, 2s, 4s, 8s, 16s (max 30s)
    -dry-run
        Discover URLs without making test requests
        Shows estimated test scope and discovered URLs
    -insecure-skip-verify
        INSECURE: Skip TLS certificate verification
        Use ONLY for testing with self-signed certificates
        Makes you vulnerable to man-in-the-middle attacks!
    -allow-private-ips
        Allow testing against private/localhost IPs
        Bypasses SSRF protection for internal testing
        Only use for services you own or have permission to test
    -ignore-robots
        Ignore robots.txt directives (use responsibly)
        Only use if you OWN the website or have explicit permission
        Bypassing robots.txt may violate terms of service
    -output string
        Output file for results (JSON format)
    -verbose
        Enable verbose logging with structured output
    -no-progress
        Disable real-time progress updates
    -compare string
        Compare performance against target (e.g., Ghost, WordPress)

AUTHENTICATION OPTIONS:
    -auth-type string
        Authentication type: basic, bearer, cookie, header
    -auth-username string
        Username for basic authentication
    -auth-password-stdin
        Read password from stdin (more secure than CLI flags)
    -auth-token-stdin
        Read bearer token from stdin (more secure than CLI flags)
    -auth-header string
        Custom header in Name:Value format

    Credentials via environment variables (RECOMMENDED):
        LOBSTER_AUTH_PASSWORD   Password for basic authentication
        LOBSTER_AUTH_TOKEN      Bearer token for authentication
        LOBSTER_AUTH_COOKIE     Cookie string in name=value format

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

    # Test with basic authentication (using environment variable)
    LOBSTER_AUTH_PASSWORD=secret lobster -url http://localhost:3000 \
        -auth-type basic -auth-username admin

    # Test with basic authentication (using stdin)
    echo "secret" | lobster -url http://localhost:3000 \
        -auth-type basic -auth-username admin -auth-password-stdin

    # Test with bearer token (using environment variable)
    LOBSTER_AUTH_TOKEN=eyJhbGc... lobster -url http://api.example.com \
        -auth-type bearer

    # Test with cookie authentication (using environment variable)
    LOBSTER_AUTH_COOKIE="session=abc123" lobster -url http://localhost:3000 \
        -auth-type cookie

    # Test with custom header (e.g., API key)
    lobster -url http://api.example.com -auth-type header \
        -auth-header "X-API-Key:your-key-here"

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
      "auth": {
        "type": "basic",
        "username": "admin",
        "password": "${LOBSTER_AUTH_PASSWORD}"
      },
      "performance_targets": {
        "requests_per_second": 100,
        "avg_response_time_ms": 50,
        "p95_response_time_ms": 100,
        "p99_response_time_ms": 200,
        "success_rate": 99.0,
        "error_rate": 1.0
      }
    }

    NOTE: Use ${VAR_NAME} syntax for environment variable substitution.
    Never store plaintext credentials in config files.

DOCUMENTATION:
    Full documentation: https://github.com/vnykmshr/lobster
    Report issues: https://github.com/vnykmshr/lobster/issues

VERSION:
    Lobster v` + version)
}

// validateRateLimit enforces safe rate limiting to prevent accidental DoS
func validateRateLimit(rate *float64) error {
	const (
		minRate  = 0.1 // Minimum allowed rate (requests per second)
		warnRate = 1.0 // Warning threshold for low rates
	)

	// Rate of 0 means no rate limiting (unlimited)
	if *rate == 0 {
		fmt.Fprintf(os.Stderr, "\nWARNING: No rate limiting enabled (rate=0)\n")
		fmt.Fprintf(os.Stderr, "This will send requests as fast as possible.\n")
		fmt.Fprintf(os.Stderr, "Make sure you have permission to test the target server.\n\n")
		return nil
	}

	// Enforce minimum rate to prevent extremely slow tests
	if *rate > 0 && *rate < minRate {
		fmt.Fprintf(os.Stderr, "\nWARNING: Rate %.2f req/s is below minimum %.2f req/s\n", *rate, minRate)
		fmt.Fprintf(os.Stderr, "Adjusting to minimum rate of %.2f req/s\n", minRate)
		fmt.Fprintf(os.Stderr, "Rationale: Extremely low rates may indicate configuration error.\n\n")
		*rate = minRate
		return nil
	}

	// Warn about very low rates and prompt for confirmation if interactive
	if *rate < warnRate {
		fmt.Fprintf(os.Stderr, "\nWARNING: Low rate limit detected (%.2f req/s)\n", *rate)
		fmt.Fprintf(os.Stderr, "This will send requests very slowly:\n")
		fmt.Fprintf(os.Stderr, "- %.2f requests per second\n", *rate)
		fmt.Fprintf(os.Stderr, "- ~%.0f requests per minute\n", *rate*60)
		fmt.Fprintf(os.Stderr, "- Test may take a long time to complete\n\n")

		// If interactive terminal, prompt for confirmation
		if isInteractiveTerminal() {
			fmt.Fprintf(os.Stderr, "Do you want to continue with this rate? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading confirmation: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				return fmt.Errorf("test canceled by user")
			}
			fmt.Fprintf(os.Stderr, "\n")
		} else {
			fmt.Fprintf(os.Stderr, "Continuing in non-interactive mode...\n\n")
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
