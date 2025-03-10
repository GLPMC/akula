package client

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gotd/td/tg"
	"golang.org/x/term"
)

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func getChannelAccessHash(ctx context.Context, api *tg.Client, channelID int64) (int64, error) {
	VerbosePrintf("Getting channel access hash...")
	result, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{
		&tg.InputChannel{
			ChannelID:  channelID,
			AccessHash: 0, // we don't know the access hash yet
		},
	})

	if err != nil {
		return 0, &ChannelError{Err: fmt.Errorf("failed to get channel: %w", err)}
	}

	chats := result.GetChats()
	if len(chats) == 0 {
		return 0, &ChannelError{Err: fmt.Errorf("channel not found or not accessible")}
	}

	channel, ok := chats[0].(*tg.Channel)
	if !ok || channel.ID != channelID {
		return 0, &ChannelError{Err: fmt.Errorf("channel not found or not accessible")}
	}

	return channel.AccessHash, nil
}

func startSpinner(ctx context.Context, message string) func() {
	spinChars := []string{"|", "/", "-", "\\"}
	i := 0
	stopCh := make(chan struct{})

	messageLength := len(message) + 5 // +5 for the space, spinner character, and some buffer

	fmt.Printf("\r%s %s", message, spinChars[0])

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				i = (i + 1) % len(spinChars)
				fmt.Printf("\r%s %s", message, spinChars[i])
			case <-stopCh:
				clearString := strings.Repeat(" ", messageLength)
				fmt.Printf("\r%s\r", clearString)
				return
			case <-ctx.Done():
				clearString := strings.Repeat(" ", messageLength)
				fmt.Printf("\r%s\r", clearString)
				return
			}
		}
	}()

	return func() {
		close(stopCh)
	}
}

func sendMessage(ctx context.Context, api *tg.Client, peer *tg.InputPeerChannel, message string) (int, error) {

	sentMsg, err := api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  message,
		RandomID: rand.Int63(),
	})

	if err != nil {
		return 0, &MessageError{Err: fmt.Errorf("failed to send message: %w", err)}
	}

	// get our message ID
	var messageID int
	if updates, ok := sentMsg.(*tg.Updates); ok {
		for _, update := range updates.Updates {
			if newMsg, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := newMsg.Message.(*tg.Message); ok {
					messageID = msg.ID
					VerbosePrintf("Our message ID: %d\n", messageID)
					break
				}
			} else if newChannelMsg, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if msg, ok := newChannelMsg.Message.(*tg.Message); ok {
					messageID = msg.ID
					VerbosePrintf("Our message ID: %d\n", messageID)
					break
				}
			}
		}
	}

	if messageID == 0 {
		fmt.Println("Warning: Could not determine our message ID")
	}

	return messageID, nil
}

func extractTextFileContent(ctx context.Context, api *tg.Client, m *tg.Message) (string, bool, error) {
	if m.Media == nil {
		return "", false, nil
	}

	doc, ok := m.Media.(*tg.MessageMediaDocument)
	if !ok {
		return "", false, nil
	}

	document, ok := doc.Document.(*tg.Document)
	if !ok {
		return "", false, nil
	}

	VerbosePrintf("Found document attachment")

	var fileName string
	var isTextFile bool

	for _, attr := range document.Attributes {
		fileAttr, ok := attr.(*tg.DocumentAttributeFilename)
		if !ok {
			continue
		}

		fileName = fileAttr.FileName
		if strings.HasSuffix(fileName, ".txt") {
			isTextFile = true
		}
		break
	}

	if !isTextFile {
		return "", false, nil
	}

	VerbosePrintf("Found text file: %s\n", fileName)

	inputDoc := &tg.InputDocument{
		ID:            document.ID,
		AccessHash:    document.AccessHash,
		FileReference: document.FileReference,
	}

	content, err := downloadDocument(ctx, api, inputDoc)
	if err != nil {
		return "", false, &MessageError{Err: fmt.Errorf("failed to download document: %w", err)}
	}

	return content, true, nil
}

func isReplyToMessage(msg tg.MessageClass, ourMessageID int) (*tg.Message, bool) {
	m, ok := msg.(*tg.Message)
	if !ok {
		return nil, false
	}

	if m.ReplyTo == nil {
		return nil, false
	}

	replyHeader, ok := m.ReplyTo.(*tg.MessageReplyHeader)
	if !ok {
		return nil, false
	}

	if replyHeader.ReplyToMsgID == ourMessageID && !m.Out {
		return m, true
	}

	return nil, false
}

func checkForReply(ctx context.Context, api *tg.Client, peer *tg.InputPeerChannel, ourMessageID int) (string, string, bool, error) {
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 20,
	})

	if err != nil {
		return "", "", false, &MessageError{Err: fmt.Errorf("failed to get message history: %w", err)}
	}

	messages, ok := history.(*tg.MessagesChannelMessages)
	if !ok {
		return "", "", false, nil
	}

	for _, msg := range messages.Messages {
		reply, isReply := isReplyToMessage(msg, ourMessageID)
		if !isReply {
			continue
		}

		response := reply.Message

		fileContent, hasFile, err := extractTextFileContent(ctx, api, reply)
		if err != nil {
			return "", "", false, err
		}

		// if we have a file, return immediately
		if hasFile {
			return response, fileContent, true, nil
		}

		return response, "\nNo records found!", true, nil
	}

	return "", "", false, nil
}

func waitForReply(ctx context.Context, api *tg.Client, peer *tg.InputPeerChannel, ourMessageID int, waitTime time.Duration) (string, string, error) {
	VerbosePrintf("Message sent successfully. Waiting for a reply for %v...\n", waitTime)

	startTime := time.Now()

	for time.Since(startTime) < waitTime {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("context canceled while waiting for response")
		case <-time.After(2 * time.Second): // poll every 2 second
			response, fileContent, found, err := checkForReply(ctx, api, peer, ourMessageID)
			if err != nil {
				return "", "", err
			}

			if found {
				return response, fileContent, nil
			}
		}
	}

	return "", "", &MessageError{Err: fmt.Errorf("no reply to our message received within the wait time")}
}

func deleteMessage(ctx context.Context, api *tg.Client, channelID, accessHash int64, messageID int) error {
	VerbosePrintf("Deleting our original message...")
	_, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
		Channel: &tg.InputChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
		ID: []int{messageID},
	})

	if err != nil {
		fmt.Printf("Warning: Failed to delete original message: %v\n", err)
	} else {
		VerbosePrintf("Original message deleted successfully")
	}

	return nil
}

func (c *Client) SendMessageAndGetResponse(ctx context.Context, channelID int64, message string, waitTime time.Duration) (string, error) {
	opCtx, cancel := context.WithTimeout(ctx, waitTime+120*time.Second)
	defer cancel()

	var stopSpinner func()
	if isTerminal() {
		stopSpinner = startSpinner(ctx, "Searching logs...")
	}

	var response string
	var fileContent string
	err := c.client.Run(opCtx, func(ctx context.Context) error {
		api := c.client.API()

		if err := c.authenticate(ctx); err != nil {
			return err
		}

		accessHash, err := getChannelAccessHash(ctx, api, channelID)
		if err != nil {
			return err
		}

		peer := &tg.InputPeerChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		}

		// send the message
		messageID, err := sendMessage(ctx, api, peer, message)
		if err != nil {
			return err
		}

		msgResponse, fileResp, err := waitForReply(ctx, api, peer, messageID, waitTime)

		response = msgResponse
		fileContent = fileResp

		return deleteMessage(ctx, api, channelID, accessHash, messageID)
	})

	if isTerminal() && stopSpinner != nil {
		stopSpinner()
		fmt.Println()
	}

	if err != nil {
		return "", err
	}

	// if we have file content, return it
	if fileContent != "" {
		return fileContent, nil
	}

	return response, nil
}
