package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

var serverPort string

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		// Use the already loaded configuration from root.go
		cfg := appConfig

		// Override with CLI flags
		if serverPort != "" {
			cfg.Port = serverPort
		}
		if logLevel != "" {
			cfg.LoggingLevel = logLevel
		}

		// Print updated configuration
		cfg.PrintConfig()

		// Determine port with proper formatting - add colon for FastHTTP
		port := cfg.Port
		if port != "" {
			port = ":" + port
		}

		handler := func(ctx *fasthttp.RequestCtx) {
			if _, err := fmt.Fprintf(ctx, "Hello from FastHTTP!"); err != nil {
				log.Error().Err(err).Msg("Failed to write response")
			}
		}
		log.Info().Msgf("Starting FastHTTP server on %s (version: %s, build time: %s)", port, appVersion, buildTime)
		if err := fasthttp.ListenAndServe(port, handler); err != nil {
			log.Error().Err(err).Msg("Error starting FastHTTP server")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&serverPort, "port", "p", "", "Port to run the server on (overrides env vars and config, default: 8080)")
}
