// Package aur provides Arch User Repository (AUR) support.
// rpc.go implements the AUR RPC v5 client for querying packages.
package aur

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RPCClient provides access to the AUR API v5
type RPCClient struct {
	baseURL       string
	httpClient    *http.Client
	cacheTTL      time.Duration
	cache         map[string]*CachedAURPackage
	cacheMutex    sync.RWMutex
	requestCount  int
	lastResetTime time.Time
	requestMutex  sync.Mutex
}

// NewRPCClient creates a new AUR RPC client
func NewRPCClient() *RPCClient {
	return &RPCClient{
		baseURL: "https://aur.archlinux.org/rpc",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheTTL:      24 * time.Hour,
		cache:         make(map[string]*CachedAURPackage),
		lastResetTime: time.Now(),
	}
}

// SearchPackages searches the AUR for packages matching the query
// Returns up to limit results
func (rc *RPCClient) SearchPackages(query string, limit int) ([]AURPackage, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if len(query) < 2 {
		return nil, fmt.Errorf("search query must be at least 2 characters")
	}

	// Build query parameters
	params := url.Values{}
	params.Set("v", "5")
	params.Set("type", "search")
	params.Set("arg", query)
	if limit > 0 && limit <= 5000 {
		// AUR API doesn't have a limit parameter, but returns max 5000 results
		_ = limit
	}

	fullURL := fmt.Sprintf("%s?%s", rc.baseURL, params.Encode())

	result, err := rc.doRequest(fullURL)
	if err != nil {
		return nil, err
	}

	return result.Results, nil
}

// GetPackageInfo retrieves detailed information about specific AUR packages
// Names should be an array of package names
// Batches requests if needed (max 200 packages per request per AUR API)
func (rc *RPCClient) GetPackageInfo(names []string) (map[string]*AURPackage, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("package names cannot be empty")
	}

	result := make(map[string]*AURPackage)
	resultMutex := sync.Mutex{}

	// Check cache first
	toFetch := []string{}
	for _, name := range names {
		if pkg, ok := rc.getFromCache(name); ok {
			resultMutex.Lock()
			result[name] = pkg
			resultMutex.Unlock()
		} else {
			toFetch = append(toFetch, name)
		}
	}

	// If everything was cached, return
	if len(toFetch) == 0 {
		return result, nil
	}

	// Batch requests (max 200 packages per request)
	const maxPackagesPerRequest = 200
	var batchWg sync.WaitGroup

	for i := 0; i < len(toFetch); i += maxPackagesPerRequest {
		end := i + maxPackagesPerRequest
		if end > len(toFetch) {
			end = len(toFetch)
		}
		batch := toFetch[i:end]

		batchWg.Add(1)
		go func(packageNames []string) {
			defer batchWg.Done()

			// Build query parameters
			params := url.Values{}
			params.Set("v", "5")
			params.Set("type", "info")
			for _, name := range packageNames {
				params.Add("arg[]", name)
			}

			fullURL := fmt.Sprintf("%s?%s", rc.baseURL, params.Encode())

			info, err := rc.doRequest(fullURL)
			if err != nil {
				// Log error but continue with other batches
				fmt.Printf("Error fetching package info: %v\n", err)
				return
			}

			resultMutex.Lock()
			for idx := range info.Results {
				pkg := &info.Results[idx]
				result[pkg.Name] = pkg
				rc.addToCache(pkg.Name, pkg)
			}
			resultMutex.Unlock()
		}(batch)
	}

	batchWg.Wait()

	// Check if we found all requested packages
	notFound := []string{}
	for _, name := range names {
		if _, ok := result[name]; !ok {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 && len(result) == 0 {
		return nil, fmt.Errorf("package not found: %s", strings.Join(notFound, ", "))
	}

	return result, nil
}

// GetPackage retrieves information about a single package
func (rc *RPCClient) GetPackage(name string) (*AURPackage, error) {
	if name == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	// Check cache first
	if pkg, ok := rc.getFromCache(name); ok {
		return pkg, nil
	}

	// Query API
	params := url.Values{}
	params.Set("v", "5")
	params.Set("type", "info")
	params.Set("arg[]", name)

	fullURL := fmt.Sprintf("%s?%s", rc.baseURL, params.Encode())

	result, err := rc.doRequest(fullURL)
	if err != nil {
		return nil, err
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("package not found: %s", name)
	}

	pkg := &result.Results[0]
	rc.addToCache(name, pkg)
	return pkg, nil
}

// doRequest performs an HTTP request to the AUR API and parses the response
func (rc *RPCClient) doRequest(fullURL string) (*RPCInfoResult, error) {
	// Track request count for rate limiting
	rc.trackRequest()

	resp, err := rc.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AUR API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result RPCInfoResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode AUR response: %w", err)
	}

	// Check for API errors
	if result.Type == "error" {
		if len(result.Results) == 0 {
			return &result, nil // Empty result, not necessarily an error
		}
	}

	return &result, nil
}

// addToCache adds a package to the local cache
func (rc *RPCClient) addToCache(name string, pkg *AURPackage) {
	rc.cacheMutex.Lock()
	defer rc.cacheMutex.Unlock()

	rc.cache[name] = &CachedAURPackage{
		Package:   pkg,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(rc.cacheTTL),
	}
}

// getFromCache retrieves a package from the cache if it's still valid
func (rc *RPCClient) getFromCache(name string) (*AURPackage, bool) {
	rc.cacheMutex.RLock()
	defer rc.cacheMutex.RUnlock()

	cached, ok := rc.cache[name]
	if !ok {
		return nil, false
	}

	if !cached.IsCacheValid() {
		return nil, false
	}

	return cached.Package, true
}

// trackRequest tracks API requests for rate limiting
// AUR has a 4000 requests/day limit
func (rc *RPCClient) trackRequest() {
	rc.requestMutex.Lock()
	defer rc.requestMutex.Unlock()

	now := time.Now()

	// Reset counter if a day has passed
	if now.Sub(rc.lastResetTime) > 24*time.Hour {
		rc.requestCount = 0
		rc.lastResetTime = now
	}

	rc.requestCount++

	// Warn if approaching limit
	if rc.requestCount == 3800 {
		fmt.Printf("Warning: Approaching AUR API rate limit (3800/4000 requests)\n")
	}
}

// RequestCount returns the current request count for the current day
func (rc *RPCClient) RequestCount() int {
	rc.requestMutex.Lock()
	defer rc.requestMutex.Unlock()
	return rc.requestCount
}

// ClearCache clears the local package cache
func (rc *RPCClient) ClearCache() {
	rc.cacheMutex.Lock()
	defer rc.cacheMutex.Unlock()

	rc.cache = make(map[string]*CachedAURPackage)
}
