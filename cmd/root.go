package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-controller.git",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Configure zerolog
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		fmt.Println("=== k8s-controller CLI ===")
		fmt.Println("Testing logging functionality...")
		fmt.Println()

		// Example logging statements
		log.Info().Msg("This is an info log")
		log.Debug().Msg("This is a debug log")
		log.Trace().Msg("This is a trace log")
		log.Warn().Msg("This is a warn log")
		log.Error().Msg("This is an error log")

		fmt.Println()
		fmt.Println("Logging test completed!")
		fmt.Println("Check the log messages above to verify zerolog is working.")
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		log.Error().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
