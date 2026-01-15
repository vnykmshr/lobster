package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// MinRate is the minimum allowed rate (requests per second).
	MinRate = 0.1
	// WarnRate is the warning threshold for low rates.
	WarnRate = 1.0
)

// ValidateRateLimit enforces safe rate limiting to prevent accidental DoS.
// It modifies the rate pointer if the rate is below minimum.
func ValidateRateLimit(rate *float64) error {
	// Rate of 0 means no rate limiting (unlimited)
	if *rate == 0 {
		fmt.Fprintf(os.Stderr, "\nWARNING: No rate limiting enabled (rate=0)\n")
		fmt.Fprintf(os.Stderr, "This will send requests as fast as possible.\n")
		fmt.Fprintf(os.Stderr, "Make sure you have permission to test the target server.\n\n")
		return nil
	}

	// Enforce minimum rate to prevent extremely slow tests
	if *rate > 0 && *rate < MinRate {
		fmt.Fprintf(os.Stderr, "\nWARNING: Rate %.2f req/s is below minimum %.2f req/s\n", *rate, MinRate)
		fmt.Fprintf(os.Stderr, "Adjusting to minimum rate of %.2f req/s\n", MinRate)
		fmt.Fprintf(os.Stderr, "Rationale: Extremely low rates may indicate configuration error.\n\n")
		*rate = MinRate
		return nil
	}

	// Warn about very low rates and prompt for confirmation if interactive
	if *rate < WarnRate {
		fmt.Fprintf(os.Stderr, "\nWARNING: Low rate limit detected (%.2f req/s)\n", *rate)
		fmt.Fprintf(os.Stderr, "This will send requests very slowly:\n")
		fmt.Fprintf(os.Stderr, "- %.2f requests per second\n", *rate)
		fmt.Fprintf(os.Stderr, "- ~%.0f requests per minute\n", *rate*60)
		fmt.Fprintf(os.Stderr, "- Test may take a long time to complete\n\n")

		// If interactive terminal, prompt for confirmation
		if IsInteractiveTerminal() {
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

// IsInteractiveTerminal checks if the program is running in an interactive terminal.
func IsInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Check if stdin is a character device (terminal) rather than a pipe or file
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
