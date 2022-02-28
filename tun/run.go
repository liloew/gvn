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
package tun

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
)

func RunCommand(filepath string, envs ...string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command(filepath)
	default:
		cmd = exec.Command("/bin/sh", filepath)
	}
	cmd.Env = os.Environ()
	for _, env := range envs {
		cmd.Env = append(cmd.Env, env)
	}
	logrus.WithFields(logrus.Fields{
		"Envs": cmd.Env,
	}).Debug("Environments")
	// calculate the mask
	if err := cmd.Run(); err != nil {
		output, _ := cmd.Output()
		logrus.WithFields(logrus.Fields{
			"script": filepath,
			"ERROR":  err,
			"OUTPUT": output,
		}).Error("Execute failed")
		return err
	} else {
		output, _ := cmd.Output()
		logrus.WithFields(logrus.Fields{
			"OUTPUT": output,
		}).Debug("Execute success")
	}
	return nil
}
