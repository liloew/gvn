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
	"bufio"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/liloew/altgvn/p2p"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("up called")
		upCommand(cmd)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringP("peers", "", "", "the peers id, splited by comma")
}

func upCommand(cmd *cobra.Command) {
	config := Config{}
	if err := viper.Unmarshal(&config); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("Unmarshal config file error")
	}
	host, err := p2p.NewPeer(config.PriKey)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR":      err,
			"PrivateKey": config.PriKey,
		}).Panic("Create peer error")
	}
	logrus.WithFields(logrus.Fields{
		"ID":    host.ID().Pretty(),
		"Addrs": host.Addrs(),
	}).Info("Peer info")
	zone := viper.GetString("protocol")
	host.SetStreamHandler(protocol.ID(zone), func(stream network.Stream) {
		logrus.WithFields(logrus.Fields{
			"LocalPeer":  stream.Conn().LocalPeer(),
			"RemotePeer": stream.Conn().RemotePeer(),
			"LocalAddr":  stream.Conn().LocalMultiaddr(),
			"RemoteAddr": stream.Conn().RemoteMultiaddr(),
			"Protocol":   stream.Protocol(),
		}).Info("handler new stream")
		// TODO: 处理特定消息
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		go readData(rw)
		go writeData(rw)
	})
	// TODO: MODECLIENT ?
	var bootstraps []string
	if MODE(viper.GetUint("mode")) == MODECLIENT {
		// start dht in server mode
		bootstraps = viper.GetStringSlice("server")
	} else {
		// DHT connect to self
		for _, addr := range host.Addrs() {
			bootstraps = append(bootstraps, fmt.Sprintf("%s/p2p/%s", addr.String(), host.ID().Pretty()))
		}
	}
	p2p.NewDHT(host, bootstraps)
	// TODO: find peers
	// peerIds := p2p.FindPeerIds()
	peerIds, _ := cmd.Flags().GetString("peers")
	logrus.WithFields(logrus.Fields{
		"PEERS": peerIds,
	}).Info("")
	streams := p2p.NewStreams(host, zone, strings.Split(peerIds, ","))
	// BEGIN: DEBUG
	for _, peerId := range strings.Split(peerIds, ",") {
		if stream, ok := streams[peerId]; ok {
			for i := 0; i < 3; i++ {
				// read until new line
				bytes := []byte(fmt.Sprintf("%d", i))
				bytes = append(bytes, "\n"...)
				stream.Write(bytes)
			}
		}
	}
	// END: DEBUG

	c := make(chan int, 1)
	<-c
}

func readData(rw *bufio.ReadWriter) {
	logrus.WithFields(logrus.Fields{
		"": "",
	}).Info("READY TO READ DATA")
	// bytes, err := rw.ReadBytes('\n')
	for {
		bytes, isPrefix, err := rw.ReadLine()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR":  err,
				"PREFIX": isPrefix,
				"BYTES":  bytes,
			}).Error("READY TO READ DATA")
			return
		}
		logrus.WithFields(logrus.Fields{
			"PREFIX": isPrefix,
			"bytes":  bytes,
			"data":   string(bytes),
		}).Info("READY TO READ DATA")
	}
}

func writeData(rw *bufio.ReadWriter) {
	logrus.WithFields(logrus.Fields{
		"": "",
	}).Info("READY TO WRITE DATA")
	for {
		// TODO:
		if _, err := rw.Write(nil); err != nil {
			// TODO:
		}
	}
}
