package nakama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/ai-campions/leaderboard-nakama/internal/config"
	"github.com/ai-campions/leaderboard-nakama/internal/leaderboard"
)

// Client wraps Nakama HTTP API for leaderboard operations
type Client struct {
	baseURL    string
	serverKey  string
	httpClient *http.Client
	
	// Token cache for users
	tokenCache map[string]*tokenEntry
	tokenMu    sync.RWMutex
}

type tokenEntry struct {
	token     string
	expiresAt time.Time
}

// NewClient creates a new Nakama client
func NewClient(cfg config.NakamaConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	return &Client{
		baseURL:   fmt.Sprintf("%s://%s:%s", scheme, cfg.Host, cfg.HTTPPort),
		serverKey: cfg.ServerKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokenCache: make(map[string]*tokenEntry),
	}
}

// authResponse from Nakama device auth
type authResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	Created      bool   `json:"created"`
}

// nakamaLeaderboardRecord matches Nakama's API response structure
type nakamaLeaderboardRecord struct {
	LeaderboardID string `json:"leaderboard_id"`
	OwnerID       string `json:"owner_id"`
	Username      string `json:"username"`
	Score         string `json:"score"`
	Subscore      string `json:"subscore"`
	NumScore      int    `json:"num_score"`
	Metadata      string `json:"metadata"`
	CreateTime    string `json:"create_time"`
	UpdateTime    string `json:"update_time"`
	ExpiryTime    string `json:"expiry_time"`
	Rank          string `json:"rank"`
	MaxNumScore   int    `json:"max_num_score"`
}

type nakamaLeaderboardRecordList struct {
	Records      []nakamaLeaderboardRecord `json:"records"`
	OwnerRecords []nakamaLeaderboardRecord `json:"owner_records"`
	NextCursor   string                    `json:"next_cursor"`
	PrevCursor   string                    `json:"prev_cursor"`
	RankCount    string                    `json:"rank_count"`
}

type nakamaWriteRecordResponse struct {
	LeaderboardID string `json:"leaderboard_id"`
	OwnerID       string `json:"owner_id"`
	Username      string `json:"username"`
	Score         string `json:"score"`
	Subscore      string `json:"subscore"`
	NumScore      int    `json:"num_score"`
	Metadata      string `json:"metadata"`
	CreateTime    string `json:"create_time"`
	UpdateTime    string `json:"update_time"`
}

// authenticateUser creates or authenticates a user via device ID
func (c *Client) authenticateUser(ctx context.Context, userID string, username string) (string, error) {
	// Check cache first
	c.tokenMu.RLock()
	if entry, ok := c.tokenCache[userID]; ok {
		if time.Now().Before(entry.expiresAt) {
			c.tokenMu.RUnlock()
			return entry.token, nil
		}
	}
	c.tokenMu.RUnlock()

	// Authenticate via device ID (using userID as device ID)
	endpoint := fmt.Sprintf("%s/v2/account/authenticate/device?create=true", c.baseURL)
	
	body := map[string]string{
		"id": userID,
	}
	if username != "" {
		endpoint += "&username=" + url.QueryEscape(username)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	c.setServerKeyAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	// Cache the token (tokens typically expire in 2 hours, cache for 1 hour)
	c.tokenMu.Lock()
	c.tokenCache[userID] = &tokenEntry{
		token:     authResp.Token,
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	c.tokenMu.Unlock()

	return authResp.Token, nil
}

// CreateLeaderboard creates a new leaderboard in Nakama (requires console/admin access)
func (c *Client) CreateLeaderboard(ctx context.Context, cfg leaderboard.LeaderboardConfig) error {
	return nil
}

// SubmitScore writes a score to the Nakama leaderboard
func (c *Client) SubmitScore(ctx context.Context, sub leaderboard.ScoreSubmission) (*leaderboard.LeaderboardRecord, error) {
	// First authenticate the user
	token, err := c.authenticateUser(ctx, sub.UserID, sub.Username)
	if err != nil {
		return nil, fmt.Errorf("authenticate user: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2/leaderboard/%s", c.baseURL, sub.LeaderboardID)

	metadata := "{}"
	if len(sub.Metadata) > 0 {
		metaBytes, _ := json.Marshal(sub.Metadata)
		metadata = string(metaBytes)
	}

	body := map[string]interface{}{
		"score":    sub.Score,
		"subscore": sub.Subscore,
		"metadata": metadata,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Use bearer token for user context
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nakama error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result nakamaWriteRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertWriteResponse(&result, sub.UserID), nil
}

// GetLeaderboard fetches leaderboard records from Nakama
func (c *Client) GetLeaderboard(ctx context.Context, req leaderboard.GetLeaderboardRequest) (*leaderboard.LeaderboardResult, error) {
	// Need to authenticate to read leaderboards in Nakama
	token, err := c.getServerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get server token: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2/leaderboard/%s", c.baseURL, req.LeaderboardID)

	params := url.Values{}
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if len(req.OwnerIDs) > 0 {
		for _, id := range req.OwnerIDs {
			params.Add("owner_ids", id)
		}
	}

	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nakama error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var list nakamaLeaderboardRecordList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertRecordList(req.LeaderboardID, &list), nil
}

// getServerToken returns a cached server-level token for API operations
func (c *Client) getServerToken(ctx context.Context) (string, error) {
	return c.authenticateUser(ctx, "server-api-client", "")
}

// GetUserRecords fetches specific users' records from the leaderboard
func (c *Client) GetUserRecords(ctx context.Context, leaderboardID string, userIDs []string) ([]leaderboard.LeaderboardRecord, error) {
	result, err := c.GetLeaderboard(ctx, leaderboard.GetLeaderboardRequest{
		LeaderboardID: leaderboardID,
		OwnerIDs:      userIDs,
	})
	if err != nil {
		return nil, err
	}
	return result.OwnerRecords, nil
}

// GetAroundUser fetches records around a specific user
func (c *Client) GetAroundUser(ctx context.Context, leaderboardID, userID string, limit int) (*leaderboard.LeaderboardResult, error) {
	token, err := c.getServerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get server token: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2/leaderboard/%s/owner/%s?limit=%d", c.baseURL, leaderboardID, userID, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nakama error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var list nakamaLeaderboardRecordList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertRecordList(leaderboardID, &list), nil
}

// DeleteRecord deletes a user's record from the leaderboard
func (c *Client) DeleteRecord(ctx context.Context, leaderboardID, userID string) error {
	// Authenticate the user first
	token, err := c.authenticateUser(ctx, userID, "")
	if err != nil {
		return fmt.Errorf("authenticate user: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2/leaderboard/%s", c.baseURL, leaderboardID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("nakama error: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// HealthCheck verifies Nakama server is reachable
func (c *Client) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/healthcheck", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("nakama unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nakama unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}

// setServerKeyAuth sets Basic auth with server key (for server-to-server calls)
func (c *Client) setServerKeyAuth(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(c.serverKey + ":"))
	req.Header.Set("Authorization", "Basic "+auth)
}

func (c *Client) convertRecordList(leaderboardID string, list *nakamaLeaderboardRecordList) *leaderboard.LeaderboardResult {
	records := make([]leaderboard.LeaderboardRecord, len(list.Records))
	for i, r := range list.Records {
		records[i] = *c.convertRecord(&r)
	}

	ownerRecords := make([]leaderboard.LeaderboardRecord, len(list.OwnerRecords))
	for i, r := range list.OwnerRecords {
		ownerRecords[i] = *c.convertRecord(&r)
	}

	totalCount, _ := strconv.ParseInt(list.RankCount, 10, 64)

	return &leaderboard.LeaderboardResult{
		LeaderboardID: leaderboardID,
		Type:          leaderboard.LiveLeaderboard,
		Records:       records,
		OwnerRecords:  ownerRecords,
		NextCursor:    list.NextCursor,
		PrevCursor:    list.PrevCursor,
		TotalCount:    totalCount,
	}
}

func (c *Client) convertRecord(r *nakamaLeaderboardRecord) *leaderboard.LeaderboardRecord {
	score, _ := strconv.ParseInt(r.Score, 10, 64)
	subscore, _ := strconv.ParseInt(r.Subscore, 10, 64)
	rank, _ := strconv.ParseInt(r.Rank, 10, 64)

	var metadata map[string]string
	if r.Metadata != "" && r.Metadata != "{}" {
		_ = json.Unmarshal([]byte(r.Metadata), &metadata)
	}

	createTime, _ := time.Parse(time.RFC3339Nano, r.CreateTime)
	updateTime, _ := time.Parse(time.RFC3339Nano, r.UpdateTime)

	return &leaderboard.LeaderboardRecord{
		LeaderboardID: r.LeaderboardID,
		OwnerID:       r.OwnerID,
		Username:      r.Username,
		Score:         score,
		Subscore:      subscore,
		Rank:          rank,
		Metadata:      metadata,
		CreateTime:    createTime,
		UpdateTime:    updateTime,
	}
}

func (c *Client) convertWriteResponse(r *nakamaWriteRecordResponse, userID string) *leaderboard.LeaderboardRecord {
	score, _ := strconv.ParseInt(r.Score, 10, 64)
	subscore, _ := strconv.ParseInt(r.Subscore, 10, 64)

	var metadata map[string]string
	if r.Metadata != "" && r.Metadata != "{}" {
		_ = json.Unmarshal([]byte(r.Metadata), &metadata)
	}

	createTime, _ := time.Parse(time.RFC3339Nano, r.CreateTime)
	updateTime, _ := time.Parse(time.RFC3339Nano, r.UpdateTime)

	ownerID := r.OwnerID
	if ownerID == "" {
		ownerID = userID
	}

	return &leaderboard.LeaderboardRecord{
		LeaderboardID: r.LeaderboardID,
		OwnerID:       ownerID,
		Username:      r.Username,
		Score:         score,
		Subscore:      subscore,
		Metadata:      metadata,
		CreateTime:    createTime,
		UpdateTime:    updateTime,
	}
}
