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

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type MODE uint

const (
	MODECLIENT MODE = iota
	MODESERVER
)

type Dev struct {
	Name string   `yaml:"name,omitempty"`
	Subs []string `yaml:"subs,omitempty"`
}

type Config struct {
	Id     string `yaml:"id,omitempty"`
	Mode   MODE   `yaml:"mode,omitempty"`
	Subnet string `yaml:"subnet,omitempty"`
	PriKey string `yaml:"priKey,omitempty"`
	PubKey string `yaml:"pubKey,omitempty"`
	Devs   []Dev  `yaml:"devs,omitempty"`
}

// initCmd represents the init command
var (
	initCmd = &cobra.Command{
		Use:   "init",
		Short: "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := os.Stat(viper.ConfigFileUsed()); err == nil {
				if force, _ := cmd.Flags().GetBool("force"); !force {
					logrus.WithFields(logrus.Fields{
						"File": viper.ConfigFileUsed(),
					}).Error("File Exists and doesn't write forcely")
					return
				}
			}
			config := Config{}
			parseConfig(*cmd, &config)
			if vip, err := yaml.Marshal(config); err == nil {
				if _, err := os.Stat(filepath.Dir(viper.ConfigFileUsed())); err != nil {
					os.MkdirAll(filepath.Dir(viper.ConfigFileUsed()), os.ModeDir)
				}
				if f, err := os.Create(viper.ConfigFileUsed()); err == nil {
					f.Write(vip)
					return
				} else {
					logrus.WithFields(logrus.Fields{
						"ERROR": err,
						"File":  viper.ConfigFileUsed(),
					}).Error("Write to config file error")
				}
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("force", "f", false, "force overide the file")
	initCmd.Flags().BoolP("server", "s", false, "server mode")
	// availabe in server mode
	initCmd.Flags().StringP("subnet", "", "192.168.1.1/24", "the CIDR subnet used in")
}

// parse the config object
func parseConfig(cmd cobra.Command, config *Config) {
	host, err := libp2p.New()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Create new peer config file error")
		return
	}
	config.Id = host.ID().Pretty()
	if priKey, err := crypto.MarshalPrivateKey(host.Peerstore().PrivKey(host.ID())); err == nil {
		pubKey, _ := crypto.MarshalPublicKey(host.Peerstore().PubKey(host.ID()))
		// TODO: !!binary leading ?
		config.PriKey = string(priKey)
		config.PubKey = string(pubKey)
	}
	config.Mode = MODECLIENT
	if server, _ := cmd.Flags().GetBool("server"); server {
		config.Mode = MODESERVER
		subnet, _ := cmd.Flags().GetString("subnet")
		config.Subnet = subnet
	}
}
