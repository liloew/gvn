package tun

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
)

// envs should be ["VAR1=VAL1", ...]
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
	}).Info("Environments")
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
