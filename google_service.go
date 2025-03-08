package mcptools

import (
	"context"
	"fmt"
	"github.com/shaharia-lab/goai/observability"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// GoogleService handles authentication and token management for Google APIs
type GoogleService struct {
	logger    observability.Logger
	config    GoogleConfig
	authToken *oauth2.Token
	oauth2Cfg *oauth2.Config
	client    *http.Client
	mu        sync.RWMutex // For thread-safe token access
}

type GoogleConfig struct {
	ClientID       string
	ClientSecret   string
	Scopes         []string
	RedirectURL    string
	AuthServerPort string
}

// NewGoogleService creates a new instance of GoogleService
func NewGoogleService(logger observability.Logger, config GoogleConfig) *GoogleService {
	return &GoogleService{
		logger: logger,
		config: config,
	}
}

// Initialize sets up OAuth2 configuration and performs CLI-based authentication
func (s *GoogleService) Initialize(ctx context.Context) error {
	s.oauth2Cfg = &oauth2.Config{
		ClientID:     s.config.ClientID,
		ClientSecret: s.config.ClientSecret,
		Scopes:       s.config.Scopes,
		Endpoint:     google.Endpoint,
		RedirectURL:  s.config.RedirectURL,
	}

	// Try to authenticate
	if err := s.authenticate(ctx); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	s.logger.Infof("+v", s.authToken)

	// Start token refresh goroutine
	go s.refreshTokenPeriodically(context.Background())

	return nil
}

// refreshTokenPeriodically handles automatic token refresh
func (s *GoogleService) refreshTokenPeriodically(ctx context.Context) {
	for {
		s.mu.RLock()
		if s.authToken == nil {
			s.mu.RUnlock()
			return
		}
		expiry := s.authToken.Expiry
		s.mu.RUnlock()

		// Calculate time until token expires
		refreshTime := time.Until(expiry) - 5*time.Minute // Refresh 5 minutes before expiry

		select {
		case <-ctx.Done():
			return
		case <-time.After(refreshTime):
			if err := s.refreshToken(ctx); err != nil {
				s.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
				}).Error("Failed to refresh token")
			}
		}
	}
}

// GetClient returns an authenticated HTTP client
func (s *GoogleService) GetClient() *http.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

// refreshToken refreshes the OAuth2 token using the refresh token
func (s *GoogleService) refreshToken(ctx context.Context) error {
	s.mu.RLock()
	if s.authToken == nil {
		s.mu.RUnlock()
		return fmt.Errorf("no token available to refresh")
	}

	currentToken := s.authToken
	s.mu.RUnlock()

	// Check if we have a refresh token
	if currentToken.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Create a new token source using the refresh token
	tokenSource := s.oauth2Cfg.TokenSource(ctx, currentToken)

	// Get new token
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	s.mu.Lock()
	s.authToken = newToken
	s.client = s.oauth2Cfg.Client(ctx, newToken)
	s.mu.Unlock()

	s.logger.WithFields(map[string]interface{}{
		"new_expiry": newToken.Expiry,
	}).Debug("Successfully refreshed OAuth token")

	return nil
}

func (s *GoogleService) startLocalServer(state string) (string, error) {
	// Create a channel to receive the authorization code
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	// Create a temporary server to handle the callback
	server := &http.Server{Addr: fmt.Sprintf(":%s", s.config.AuthServerPort)}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Verify state to prevent CSRF
		if r.URL.Query().Get("state") != state {
			errorChan <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Get the authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errorChan <- fmt.Errorf("no code received")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		// Send success message to browser
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Authorization Successful</h1><p>You can close this window now.</p>")

		codeChan <- code
	})

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	// Ensure server is closed
	defer server.Close()

	// Wait for either code or error
	select {
	case code := <-codeChan:
		return code, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(2 * time.Minute):
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

// Modify the authenticate method
func (s *GoogleService) authenticate(ctx context.Context) error {
	// Generate a random state
	state := fmt.Sprintf("%d", time.Now().UnixNano())

	// Generate authorization URL
	authURL := s.oauth2Cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Printf("Opening browser to visit the authorization URL:\n%v\n\n", authURL)

	// Open the URL in the default browser
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", authURL).Start()
	case "windows":
		err = exec.Command("cmd", "/c", "start", authURL).Start()
	case "darwin":
		err = exec.Command("open", authURL).Start()
	default:
		fmt.Printf("Please open this URL in your browser:\n%v\n\n", authURL)
	}
	if err != nil {
		fmt.Printf("Failed to open browser automatically. Please open this URL manually:\n%v\n\n", authURL)
	}

	// Start local server and wait for the code
	authCode, err := s.startLocalServer(state)
	if err != nil {
		return fmt.Errorf("failed to get authorization code: %w", err)
	}

	// Exchange auth code for token
	token, err := s.oauth2Cfg.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("unable to exchange auth code: %w", err)
	}

	s.mu.Lock()
	s.authToken = token
	s.client = s.oauth2Cfg.Client(ctx, token)
	s.mu.Unlock()

	return nil
}
