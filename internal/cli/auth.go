package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vnykmshr/lobster/internal/domain"
)

// BuildAuthConfig builds authentication configuration from environment variables and stdin.
// Credentials are read from:
// 1. Environment variables (LOBSTER_AUTH_PASSWORD, LOBSTER_AUTH_TOKEN, LOBSTER_AUTH_COOKIE)
// 2. Stdin when --auth-password-stdin or --auth-token-stdin flags are used
// CLI flags for credentials are intentionally not supported to prevent exposure in process lists.
func BuildAuthConfig(opts *ConfigOptions) (*domain.AuthConfig, error) {
	// Validate stdin flags are mutually exclusive (can only read one value from stdin)
	if opts.AuthPasswordStdin && opts.AuthTokenStdin {
		return nil, fmt.Errorf("--auth-password-stdin and --auth-token-stdin are mutually exclusive")
	}

	// Get credentials from environment variables
	password := os.Getenv("LOBSTER_AUTH_PASSWORD")
	token := os.Getenv("LOBSTER_AUTH_TOKEN")
	cookie := os.Getenv("LOBSTER_AUTH_COOKIE")

	// Read from stdin if requested (overrides env vars)
	if opts.AuthPasswordStdin {
		stdinPassword, err := ReadSecretFromStdin("password")
		if err != nil {
			return nil, err
		}
		password = stdinPassword
	}

	if opts.AuthTokenStdin {
		stdinToken, err := ReadSecretFromStdin("token")
		if err != nil {
			return nil, err
		}
		token = stdinToken
	}

	// Check if any auth configuration is provided
	hasAuth := opts.AuthType != "" || opts.AuthUsername != "" || opts.AuthHeader != "" ||
		password != "" || token != "" || cookie != ""

	if !hasAuth {
		return nil, nil
	}

	authCfg := &domain.AuthConfig{
		Type:     opts.AuthType,
		Username: opts.AuthUsername,
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
	if opts.AuthHeader != "" {
		parts := strings.SplitN(opts.AuthHeader, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("invalid auth header format: expected 'Name:Value', got %q", opts.AuthHeader)
		}
		authCfg.Headers = make(map[string]string)
		authCfg.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return authCfg, nil
}

// ReadSecretFromStdin reads a single line from stdin for secure credential input.
// Returns an error if stdin is empty or closed without data.
func ReadSecretFromStdin(name string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading %s from stdin: %w", name, err)
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", fmt.Errorf("%s from stdin is empty", name)
	}
	return trimmed, nil
}
