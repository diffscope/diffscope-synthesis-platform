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

package diffsinger

import (
	"time"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("diffsinger.phoneme_cleanup_timeout_msec", 600000)
	viper.SetDefault("diffsinger.phoneme_cleanup_interval_msec", 60000)
	viper.SetDefault("diffsinger.phoneme_custom_worker_count", 16)
	viper.SetDefault("diffsinger.phoneme_custom_worker_timeout_msec", 5000)
	viper.SetDefault("diffsinger.inference_cleanup_timeout_msec", 600000)
	viper.SetDefault("diffsinger.inference_cleanup_interval_msec", 60000)
}

func getPhonemeCleanupTimeout() time.Duration {
	return time.Duration(viper.GetInt("diffsinger.phoneme_cleanup_timeout_msec")) * time.Millisecond
}

func getPhonemeCleanupInterval() time.Duration {
	return time.Duration(viper.GetInt("diffsinger.phoneme_cleanup_interval_msec")) * time.Millisecond
}

func getPhonemeCustomWorkerCount() int {
	return viper.GetInt("diffsinger.phoneme_custom_worker_count")
}

func getPhonemeCustomWorkerTimeout() time.Duration {
	return time.Duration(viper.GetInt("diffsinger.phoneme_custom_worker_timeout_msec")) * time.Millisecond
}

func getInferenceCleanupTimeout() time.Duration {
	return time.Duration(viper.GetInt("diffsinger.inference_cleanup_timeout_msec")) * time.Millisecond
}

func getInferenceCleanupInterval() time.Duration {
	return time.Duration(viper.GetInt("diffsinger.inference_cleanup_interval_msec")) * time.Millisecond
}
