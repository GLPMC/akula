package client

import (
	"github.com/glpmc/akula/config"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gotd/td/session"
)

func checkAndCleanupSession() {
	sessionPath := config.GetSessionPath()

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		fmt.Printf("Error reading session file: %v. Will create a new one.\n", err)
		os.Remove(sessionPath)
		return
	}

	var jsonObj interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		fmt.Printf("Session file is corrupted: %v. Will create a new one.\n", err)
		os.Remove(sessionPath)
		return
	}

	// a small session file is likely corrupted - there are reported issues with telegram truncating it (wtf)
	if len(data) < 10 {
		fmt.Println("Session file is too small, likely corrupted. Will create a new one.")
		os.Remove(sessionPath)
	}
}

func createSessionStorage() session.Storage {
	if sessionBase64 := os.Getenv("AKULA_SESSION"); sessionBase64 != "" {
		VerbosePrintf("Using session data from AKULA_SESSION environment variable")

		sessionData, err := base64.StdEncoding.DecodeString(sessionBase64)
		if err != nil {
			fmt.Printf("Error decoding AKULA_SESSION: %v. Falling back to file storage.\n", err)
		} else {
			storage := &session.StorageMemory{}
			if err := storage.StoreSession(context.Background(), sessionData); err != nil {
				fmt.Printf("Error loading session data: %v. Falling back to file storage.\n", err)
			} else {
				return storage
			}
		}
	}

	// fall back to file storage if environment variable is not set or invalid
	checkAndCleanupSession()

	sessionStorage := &session.FileStorage{Path: config.GetSessionPath()}

	VerbosePrintf("Using session file: %s\n", config.GetSessionPath())

	if _, err := os.Stat(config.GetSessionPath()); os.IsNotExist(err) {
		fmt.Println("Session file does not exist, will create a new one")
	} else {
		VerbosePrintf("Found existing session file")
	}

	return sessionStorage
}
