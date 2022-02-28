/*
Copyright © 2022 lilo <luolee.me@gmail.com>

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

	"github.com/sevlyar/go-daemon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "run gvn daemon",
	Long:  `Run gvn as a deamon service, all the args must be consistent with up sub-command`,
	Run: func(cmd *cobra.Command, args []string) {

		pidFile := filepath.Join(os.TempDir(), "gvn.pid")
		logFile := filepath.Join(os.TempDir(), "gvn.log")
		pwd, err := os.Getwd()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
			}).Fatal("Current dir doesn't accessable")
		}
		cntxt := &daemon.Context{
			PidFileName: pidFile,
			PidFilePerm: 0644,
			LogFileName: logFile,
			LogFilePerm: 0640,
			WorkDir:     pwd,
			Umask:       027,
			Args:        args,
		}
		d, err := cntxt.Reborn()
		if err != nil {
			logrus.Fatal("Unable to run gvn as daemon" + err.Error())
		}
		if d != nil {
			return
		}
		defer cntxt.Release()
		upCommand(cmd)
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.Flags().Uint32P("pid", "p", 0, "the gvn process pid to be stop")
}
