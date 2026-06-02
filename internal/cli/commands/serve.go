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

package commands

import (
	"diffscope-synthesis-platform/internal/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewServeCommand() (*cobra.Command, error) {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start DSSP service",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.StartServer()
		},
	}

	serveCmd.Flags().String("host", "127.0.0.1", "Host to bind")
	serveCmd.Flags().Int("port", 13711, "Port to bind")

	if err := viper.BindPFlag("host", serveCmd.Flags().Lookup("host")); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag("port", serveCmd.Flags().Lookup("port")); err != nil {
		return nil, err
	}

	return serveCmd, nil
}
