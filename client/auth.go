package client

import (
	"context"
	"fmt"
	"time"

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

	err := c.client.Run(authCtx, func(ctx context.Context) error {
		c.api = c.client.API()

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

