/**************************************************************************
 * DiffScope Synthesis Platform                                           *
 * Copyright (C) 2026 Team OpenVPI                                        *
 *                                                                        *
 * This program is free software: you can redistribute it and/or modify   *
 * it under the terms of the GNU General Public License as published by   *
 * the Free Software Foundation, either version 3 of the License, or      *
 * (at your option) any later version.                                    *
 *                                                                        *
 * This program is distributed in the hope that it will be useful,        *
 * but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 * GNU General Public License for more details.                           *
 *                                                                        *
 * You should have received a copy of the GNU General Public License      *
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. *
 **************************************************************************/

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"diffscope-synthesis-platform/internal/appinfo"
	"diffscope-synthesis-platform/internal/cli/commands"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Execute() error {
	rootCmd, err := newRootCommand()
	if err != nil {
		return err
	}
	return rootCmd.Execute()
}

func newRootCommand() (*cobra.Command, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	defaultRootDir := filepath.Join(homeDir, ".dssp")

	rootCmd := &cobra.Command{
		Use:           "dssp",
		Short:         appinfo.ApplicationName + " CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeConfig(cmd, defaultRootDir)
		},
	}
	rootCmd.Version = appinfo.ApplicationSemver

	flags := rootCmd.PersistentFlags()
	flags.String("config-dir", filepath.Join(defaultRootDir, "config"), "Directory that contains config file")
	flags.String("package-dir", filepath.Join(defaultRootDir, "packages"), "Directory for packages")
	flags.String("log-dir", filepath.Join(defaultRootDir, "logs"), "Directory for logs")
	flags.String("cache-dir", filepath.Join(defaultRootDir, "cache"), "Directory for cache")
	flags.Bool("verbose", false, "Enable verbose logging")

	rootCmd.Flags().BoolP("version", "v", false, "Print version")

	if err := viper.BindPFlag("package_dir", flags.Lookup("package-dir")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("log_dir", flags.Lookup("log-dir")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("cache_dir", flags.Lookup("cache-dir")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("verbose", flags.Lookup("verbose")); err != nil {
		return nil, err
	}

	serveCmd, err := commands.NewServeCommand()
	if err != nil {
		return nil, err
	}

	listDevicesCmd, err := commands.NewListDevicesCommand(PrintDevices)
	if err != nil {
		return nil, err
	}

	rootCmd.AddCommand(
		serveCmd,
		listDevicesCmd,
	)

	return rootCmd, nil
}

func initializeConfig(cmd *cobra.Command, defaultRootDir string) error {
	configDir, err := cmd.Flags().GetString("config-dir")
	if err != nil {
		return err
	}

	viper.SetEnvPrefix("DSSP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("host", "127.0.0.1")
	viper.SetDefault("port", 13711)
	viper.SetDefault("package_dir", filepath.Join(defaultRootDir, "packages"))
	viper.SetDefault("log_dir", filepath.Join(defaultRootDir, "logs"))
	viper.SetDefault("cache_dir", filepath.Join(defaultRootDir, "cache"))
	viper.SetDefault("verbose", false)

	viper.SetConfigName("config")
	viper.AddConfigPath(configDir)

	if err := viper.ReadInConfig(); err != nil {
		var configNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configNotFound) {
			return fmt.Errorf("read config file: %w", err)
		}
	}

	return nil
}
