package cli

import (
	"fmt"
	"os"
	"strings"
)

// PrintWarningBox prints a formatted warning box to stderr.
// The box has a consistent width and styling for security/ethical warnings.
func PrintWarningBox(title string, lines []string) {
	const boxWidth = 68 // Inner content width

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "╔%s╗\n", strings.Repeat("═", boxWidth+2))
	fmt.Fprintf(os.Stderr, "║ %-*s ║\n", boxWidth, CenterText(title, boxWidth))
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

// CenterText centers text within a given width.
func CenterText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

// ShowHelpMessage prints the help message to stdout.
// The version parameter should be passed from the main package's version variable.
func ShowHelpMessage(version string) {
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
    Full documentation: https://github.com/1mb-dev/lobster
    Report issues: https://github.com/1mb-dev/lobster/issues

VERSION:
    Lobster v` + version)
}
