package commands

import (
	"akula/client"
	"akula/config"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var akula_channel_id int64 = 1943303299
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "akula [term]",
	Short: "Akula - Search for stealer log data",
	Long:  `A command line interface for interacting with the BF Repo Akula bot, which is a Telegram bot for searching and retrieving stealer log data.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {

	var (
		apiID       int
		apiHash     string
		phoneNumber string
		waitTime    int
	)

	rootCmd.Flags().IntVar(&apiID, "api-id", 0, "Telegram API ID")
	rootCmd.Flags().StringVar(&apiHash, "api-hash", "", "Telegram API Hash")
	rootCmd.Flags().StringVar(&phoneNumber, "phone", "", "Phone number for Telegram login")
	rootCmd.Flags().IntVar(&waitTime, "wait", 30, "Time to wait for response in seconds")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}

// verbosePrintf prints a message only if verbose mode is enabled
func verbosePrintf(format string, a ...interface{}) {
	if verbose {
		fmt.Printf(format, a...)
	}
}

func promptTGCredentials() (int, string, string) {
	var apiID int
	var apiHash string
	var phoneNumber string

	fmt.Print("Please enter your Telegram API ID: ")
	fmt.Scanln(&apiID)

	fmt.Print("Please enter your Telegram API Hash: ")
	fmt.Scanln(&apiHash)

	fmt.Print("Please enter your Telegram phone number: ")
	fmt.Scanln(&phoneNumber)

	return apiID, apiHash, phoneNumber
}

func runSearch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if cfg == nil {
		cfg = &config.Config{}
	}

	apiID, _ := cmd.Flags().GetInt("api-id")
	apiHash, _ := cmd.Flags().GetString("api-hash")
	phoneNumber, _ := cmd.Flags().GetString("phone")
	waitTime, _ := cmd.Flags().GetInt("wait")
	verbose, _ = cmd.Flags().GetBool("verbose")

	if apiID != 0 {
		cfg.TGAPIID = apiID
	}
	if apiHash != "" {
		cfg.TGAPIHash = apiHash
	}
	if phoneNumber != "" {
		cfg.PhoneNumber = phoneNumber
	}

	if cfg.TGAPIID == 0 || cfg.TGAPIHash == "" || cfg.PhoneNumber == "" {
		cfg.TGAPIID, cfg.TGAPIHash, cfg.PhoneNumber = promptTGCredentials()
	}

	if cfg.TGAPIID == 0 || cfg.TGAPIHash == "" {
		return fmt.Errorf("missing required Telegram credentials. Use flags or provide them when prompted")
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	query := strings.Join(args, " ")
	verbosePrintf("Sending message to channel %d: %s\n", akula_channel_id, query)

	response, err := client.RunSearch(ctx, cfg, akula_channel_id, query, time.Duration(waitTime)*time.Second)
	if err != nil {
		return fmt.Errorf("error sending message and getting response: %w", err)
	}

	fmt.Println(response)
	return nil
}
