package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/glpmc/akula/config"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

func (c *Client) authenticate(ctx context.Context) error {
	status, err := c.client.Auth().Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth status: %w", err)
	}

	if !status.Authorized {
		flow := auth.NewFlow(
			auth.Constant(c.cfg.PhoneNumber, "", auth.CodeAuthenticatorFunc(
				func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
					fmt.Print("Enter the code sent to your device: ")
					var code string
					fmt.Scanln(&code)
					return code, nil
				},
			)),
			auth.SendCodeOptions{},
		)

		if err := c.client.Auth().IfNecessary(ctx, flow); err != nil {
			return &AuthError{Err: fmt.Errorf("failed to authenticate: %w", err)}
		}
	}

	return nil
}

func (c *Client) SimpleAuth(ctx context.Context) error {
	authCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	hasSession := false
	if os.Getenv("AKULA_SESSION") != "" {
		hasSession = true
	} else {
		sessionPath := config.GetSessionPath()
		if _, err := os.Stat(sessionPath); err == nil {
			data, err := os.ReadFile(sessionPath)
			if err == nil && len(data) > 10 {
				var jsonObj any
				if err := json.Unmarshal(data, &jsonObj); err == nil {
					hasSession = true
				}
			}
		}
	}

	err := c.client.Run(authCtx, func(ctx context.Context) error {
		c.api = c.client.API()

		if hasSession {
			status, err := c.client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to get auth status: %w", err)
			}

			if status.Authorized {
				VerbosePrintf("Already authorized using existing session\n")
				return nil
			}
		}

		if err := c.authenticate(ctx); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return &AuthError{Err: fmt.Errorf("authentication failed: %w", err)}
	}

	c.initialized = true
	c.connected = true
	return nil
}
