/*
Copyright © 2022 liluo <luolee.me@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "gvn",
		Short: "gvn - a golang implementation VPN aims on distributed work environments",
		Long:  ``,
	}
	cfgFile string
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	filename := filepath.Join(os.TempDir(), "gvn.log")
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		logrus.Fatal(err)
	}
	fmt.Fprintln(os.Stderr, "Log file - "+file.Name())
	logrus.SetOutput(file)

	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.gvn.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		cfgFile = filepath.Join(home, "gvn.yaml")
		rootCmd.Flags().Set("config", cfgFile)
	}
	viper.SetConfigFile(cfgFile)

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err == nil {
		// PASS
	} else {
	}
}
