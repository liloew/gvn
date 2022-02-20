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
	"strings"

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

type Device struct {
	Name    string   `yaml:"name,omitempty"`
	Vip     string   `yaml:"vip,omitempty"`
	Mtu     uint     `yaml:"mtu,omitempty"`
	Subnets []string `yaml:"subnets,omitempty"`
}

type Config struct {
	Id     string `yaml:"id,omitempty"`
	Port   uint   `yaml:"port,omitempty"`
	Mode   MODE   `yaml:"mode,omitempty"`
	Server string `yaml:"server,omitempty"`
	Dev    Device `yaml:"dev,omitempty"`
	Protol string `yaml:"protocol"`
	PriKey string `yaml:"priKey,omitempty"`
	PubKey string `yaml:"pubKey,omitempty"`
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
	initCmd.Flags().UintP("port", "", 6543, "the port all the other nodes connect to")
	initCmd.Flags().StringP("protocol", "", "/gvn/1.0.0", "the protocol support currently")
	initCmd.Flags().StringP("devname", "", "", "the TUN device name, recommend using utun[\\d] for cross platform, utun3 for example")
	initCmd.Flags().StringP("subnets", "", "", "the subnets traffice through this node")
	initCmd.Flags().UintP("mtu", "", 1500, "the MUT will be used in TUN device")
	// availabe in server mode
	initCmd.Flags().StringP("vip", "", "192.168.1.1/24", "the CIDR subnet used in server, all clients in the same subnet with a fake DHCP")
}

// parse the config object
func parseConfig(cmd cobra.Command, config *Config) {
	var dev Device
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
		vip, _ := cmd.Flags().GetString("vip")
		dev.Vip = vip
	} else {
		// TODO: optimizer
		config.Server = "SERVER ADDR HERE"
	}
	if port, err := cmd.Flags().GetUint("port"); err == nil {
		config.Port = port
	}
	if devName, err := cmd.Flags().GetString("devname"); err == nil {
		dev.Name = devName
	}
	if mtu, err := cmd.Flags().GetUint("mtu"); err == nil {
		dev.Mtu = mtu
	}
	if subnets, err := cmd.Flags().GetString("subnets"); err == nil {
		// sbs := make([]string, 0)
		// sbs = append(sbs, strings.Split(subnets, ",")...)
		dev.Subnets = append(dev.Subnets, strings.Split(subnets, ",")...)
	}
	if protol, err := cmd.Flags().GetString("protocol"); err == nil {
		config.Protol = protol
	}
	config.Dev = dev
}
