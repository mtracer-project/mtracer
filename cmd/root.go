/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mtrace-project/mtrace/configuration"
	"github.com/mtrace-project/mtrace/domain"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ConfigFile string
	Config     configuration.AppConfig
	version    = "v0.0.1"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   domain.CLI_NAME,
	Short: fmt.Sprintf("%s is a tool for testing thorugh trace validation", domain.CLI_NAME),
	Long: fmt.Sprintf(`%s is a CLI tool designed to facilitate testing through trace validation. 
	It allows users to define tests in YAML files, execute them, and validate the results against expected traces.`, domain.CLI_NAME),
	Version:           version,
	PersistentPreRunE: initConfig,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&Config.Directory, "dir", "d", "", "Directory to apply the command to")
	rootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", "", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVarP(&Config.Verbose, "verbose", "V", false, "Enable more verbose output for debugging purposes")
	rootCmd.PersistentFlags().BoolVarP(&Config.Quiet, "quiet", "q", false, "Suppress result output, only show errors and warnings")
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func initConfig(cmd *cobra.Command, args []string) error {
	// LEVEL 6 - default configuration
	viper.SetDefault("directory", "")
	viper.SetDefault("verbose", false)
	viper.SetDefault("quiet", false)
	viper.SetDefault("backend_type", "openobserve")
	viper.SetDefault("openobserve.base_url", "http://localhost:5080")
	viper.SetDefault("openobserve.org_name", "default")
	viper.SetDefault("openobserve.stream_name", "default")
	viper.SetDefault("openobserve.username", "admin@example.com")
	viper.SetDefault("openobserve.password", "admin")
	viper.SetDefault("jaeger.base_url", "http://localhost:16686")

	// LEVEL 3 - environment variables setup
	viper.SetEnvPrefix("MTRACE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// LEVEL 2 - command-line flags binding
	if flag := cmd.Flags().Lookup("verbose"); flag != nil {
		if err := viper.BindPFlag("verbose", flag); err != nil {
			return fmt.Errorf("error binding verbose flag: %w", err)
		}
	}
	if flag := cmd.Flags().Lookup("dir"); flag != nil {
		if err := viper.BindPFlag("directory", flag); err != nil {
			return fmt.Errorf("error binding dir flag: %w", err)
		}
	}
	if flag := cmd.Flags().Lookup("quiet"); flag != nil {
		if err := viper.BindPFlag("quiet", flag); err != nil {
			return fmt.Errorf("error binding quiet flag: %w", err)
		}
	}

	// LEVEL 4 - configuration file
	if ConfigFile != "" {
		viper.SetConfigFile(ConfigFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("mtrace")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshalling into the AppConfig struct
	if err := viper.Unmarshal(&Config); err != nil {
		return fmt.Errorf("error while unmarshaling config: %w", err)
	}

	// Setup logger
	opts := &tint.Options{
		Level:      slog.LevelWarn,
		TimeFormat: time.Kitchen,
		NoColor:    false,
	}
	if Config.Verbose {
		opts.Level = slog.LevelDebug
		opts.TimeFormat = time.RFC3339Nano
	}
	logger := slog.New(tint.NewHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	return nil
}
