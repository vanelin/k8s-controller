package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/vanelin/k8s-controller.git/pkg/common/config"
)

var (
	logLevel   string
	appConfig  config.Config
	appVersion = "dev" // This will be set during build time via ldflags
)

// parseLogLevel converts string log level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// configureLogger sets up zerolog with the specified log level
func configureLogger(level zerolog.Level) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch level {
	case zerolog.TraceLevel:
		zerolog.CallerFieldName = "caller"
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05.000",
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
		}).With().Caller().Logger()
	case zerolog.DebugLevel:
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05",
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.MessageFieldName,
			},
		})
	default:
		log.Logger = log.Output(os.Stderr)
	}

	zerolog.SetGlobalLevel(level)
}

// loadConfiguration loads environment variables using Viper
func loadConfiguration() error {
	configPath := config.GetConfigPath()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	appConfig = cfg
	return nil
}

// getLogLevel returns the log level from config or CLI flag
func getLogLevel() string {
	// If CLI flag is set, it takes precedence
	if logLevel != "" {
		return logLevel
	}
	// Otherwise use config value
	if appConfig.LoggingLevel != "" {
		return appConfig.LoggingLevel
	}
	// Default fallback
	return "info"
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "k8s-controller",
	Short:   "A Kubernetes controller with FastHTTP server (version: " + appVersion + ")",
	Long:    `A Go-based Kubernetes controller with structured logging, environment configuration using Viper, and a FastHTTP server.`,
	Version: appVersion,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load configuration first
		if err := loadConfiguration(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		}

		// Configure logging with resolved log level
		level := parseLogLevel(getLogLevel())
		configureLogger(level)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("=== k8s-controller CLI ===")
		fmt.Printf("Version: %s\n", appVersion)
		fmt.Printf("Current log level: %s\n", getLogLevel())
		fmt.Println()

		// Update config with actual log level for display
		appConfig.LoggingLevel = getLogLevel()

		// Print configuration
		appConfig.PrintConfig()
		fmt.Println()

		// Example logging statements
		log.Info().Msg("This is an info log")
		log.Debug().Msg("This is a debug log")
		log.Trace().Msg("This is a trace log")
		log.Warn().Msg("This is a warn log")
		log.Error().Msg("This is an error log")

		fmt.Println()
		fmt.Println("Check the log messages above to verify zerolog is working.")
	},
}

func init() {
	// Add log-level flag
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "", "Set the logging level (trace, debug, info, warn, error). Overrides LOGGING_LEVEL from config (default: info)")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		log.Error().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
