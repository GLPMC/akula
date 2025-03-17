package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/glpmc/akula/config"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
)

type MessageData struct {
	ChannelTitle    string `json:"channel_title"`
	ChannelUsername string `json:"channel_username"`
	MessageID       int    `json:"message_id"`
	Date            string `json:"date"`
	Message         string `json:"message"`
	URL             string `json:"url"`
}

type Client struct {
	cfg         *config.Config
	client      *telegram.Client
	api         *tg.Client
	mutex       sync.Mutex
	initialized bool
	connected   bool
}

type (
	AuthError struct {
		Err error
	}

	ChannelError struct {
		Err error
	}

	MessageError struct {
		Err error
	}
)

func (e AuthError) Error() string {
	return fmt.Sprintf("authentication error: failed to authenticate with Telegram API: %v", e.Err)
}

func (e ChannelError) Error() string {
	return fmt.Sprintf("channel error: failed to access or retrieve channel information: %v", e.Err)
}

func (e MessageError) Error() string {
	return fmt.Sprintf("message error: failed to process or retrieve message: %v", e.Err)
}

var IsVerbose bool

func SetVerbose(verbose bool) {
	IsVerbose = verbose
}

func VerbosePrintf(format string, a ...any) {
	if IsVerbose {
		fmt.Printf(format, a...)
	}
}

func NewClient(cfg *config.Config) (*Client, error) {
	sessionStorage := createSessionStorage()
	
	hasSession := false

	if os.Getenv("AKULA_SESSION") != "" {
		hasSession = true
		VerbosePrintf("Using session from AKULA_SESSION environment variable\n")
	} else {
		sessionPath := config.GetSessionPath()
		if _, err := os.Stat(sessionPath); err == nil {
			data, err := os.ReadFile(sessionPath)
			if err == nil && len(data) > 10 {
				var jsonObj any
				if err := json.Unmarshal(data, &jsonObj); err == nil {
					hasSession = true
					VerbosePrintf("Using existing session file: %s\n", sessionPath)
				}
			}
		}
	}

	var apiID int
	var apiHash string
	
	// if we have a session we can use actual placeholder values
	// the gotd/td library requires these parameters along with session file even if the tgapi stuff is wrong
	if hasSession {
		apiID = 1           // placeholder api id
		apiHash = "abcdef"  // placeholder api hash
		VerbosePrintf("Using existing session for authentication (with placeholder API credentials)\n")
	} else {
		apiID = cfg.TGAPIID
		apiHash = cfg.TGAPIHash
		VerbosePrintf("No existing session found, will authenticate with API ID and hash\n")
	}

	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: sessionStorage,
		RetryInterval:  5 * time.Second,
		MaxRetries:     5,
	})

	return &Client{
		cfg:         cfg,
		client:      client,
		mutex:       sync.Mutex{},
		initialized: false,
		connected:   false,
	}, nil
}

func (c *Client) RunSearch(ctx context.Context, channelID int64, searchTerm string, waitTime time.Duration) (string, error) {
	searchString := searchTerm
	if !strings.HasPrefix(searchTerm, "/") {
		searchString = fmt.Sprintf("/s %s", searchTerm)
	}

	VerbosePrintf("Searching for term: %s\n", searchTerm)
	return c.SendMessageAndGetResponse(ctx, channelID, searchString, waitTime)
}

func (c *Client) Run(ctx context.Context, searchTerm string) error {
	if err := c.SimpleAuth(ctx); err != nil {
		return fmt.Errorf("error authenticating: %w", err)
	}

	if searchTerm != "" {
		fmt.Printf("Searching for term: %s\n", searchTerm)
	}

	return nil
}

func RunClient(ctx context.Context, cfg *config.Config, searchTerm string, channelID int64) error {
	client, err := NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return client.Run(ctx, searchTerm)
}

func RunSearch(ctx context.Context, cfg *config.Config, channelID int64, searchTerm string, waitTime time.Duration) (string, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	return client.RunSearch(ctx, channelID, searchTerm, waitTime)
}
