package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	logLevel string
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

	if level == zerolog.TraceLevel {
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
	} else if level == zerolog.DebugLevel {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05",
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.MessageFieldName,
			},
		})
	} else {
		log.Logger = log.Output(os.Stderr)
	}

	zerolog.SetGlobalLevel(level)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-controller.git",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := parseLogLevel(logLevel)
		configureLogger(level)
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("=== k8s-controller CLI ===")
		fmt.Printf("Current log level: %s\n", logLevel)
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
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "Set the logging level (trace, debug, info, warn, error)")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		log.Error().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
