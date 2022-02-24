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
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		var pid int
		if p, err := cmd.Flags().GetUint32("pid"); err == nil && pid > 0 {
			pid = int(p)
		} else {
			filename := filepath.Join(os.TempDir(), "gvn.pid")
			if buff, err := os.ReadFile(filename); err == nil {
				if p, err := strconv.Atoi(string(buff)); err == nil {
					pid = p
				}
			}
		}
		logrus.WithFields(logrus.Fields{
			"PID": pid,
		}).Info("kill gvn process")
		syscall.Kill(pid, syscall.SIGINT)
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().Uint32P("pid", "p", 0, "the gvn process pid to be stop")
}
