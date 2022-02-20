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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/liloew/altgvn/dhcp"
	"github.com/liloew/altgvn/p2p"
	"github.com/liloew/altgvn/tun"
	"github.com/sirupsen/logrus"
	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water/waterutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// upCmd represents the up command
var (
	upCmd = &cobra.Command{
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
	mainDev tun.Device
)

func init() {
	rootCmd.AddCommand(upCmd)
	upCmd.Flags().StringP("peers", "", "", "the peers id, splited by comma")
}

func upCommand(cmd *cobra.Command) {
	devChan := make(chan tun.Device, 1)
	config := Config{}
	if err := viper.Unmarshal(&config); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("Unmarshal config file error")
	}
	host, err := p2p.NewPeer(config.PriKey, config.Port)
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
	// BEGIN: STREAM HANDLER
	host.SetStreamHandler(protocol.ID(zone), func(stream network.Stream) {
		logrus.WithFields(logrus.Fields{
			"LocalPeer":  stream.Conn().LocalPeer(),
			"RemotePeer": stream.Conn().RemotePeer(),
			"LocalAddr":  stream.Conn().LocalMultiaddr(),
			"RemoteAddr": stream.Conn().RemoteMultiaddr(),
			"Protocol":   stream.Protocol(),
		}).Info("handler new stream")
		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
		go readData(stream, rw)
	})
	// END: STREAM HANDLER
	var bootstraps []string
	if MODE(viper.GetUint("mode")) == MODECLIENT {
		// start dht in server mode
		bootstraps = viper.GetStringSlice("server")
	} else {
		// DHT connect to self
		for _, addr := range host.Addrs() {
			bootstraps = append(bootstraps, fmt.Sprintf("%s/p2p/%s", addr.String(), host.ID().Pretty()))
		}
		dhcp.NewRPCServer(host, zone, viper.GetString("dev.vip"))
		// auto config in server mode
		devChan <- tun.Device{
			Name:      viper.GetString("dev.name"),
			Ip:        viper.GetString("dev.vip"),
			Mtu:       viper.GetInt("dev.mtu"),
			Subnets:   viper.GetStringSlice("dev.subnets"),
			ServerVIP: viper.GetString("dev.vip"),
			Port:      viper.GetUint("port"),
		}
	}
	p2p.NewDHT(host, zone, bootstraps)
	// BEGIN: DHCP for client mode
	if MODE(viper.GetUint("mode")) == MODECLIENT {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			for _ = range ticker.C {
				req := dhcp.Request{
					Id:      viper.GetString("id"),
					Name:    viper.GetString("dev.name"),
					Subnets: viper.GetStringSlice("dev.subnets"),
				}
				res := dhcp.NewRPCClient(host, zone, viper.GetString("server"), req)
				logrus.WithFields(logrus.Fields{
					"RES": res,
				}).Info("DHCP - Got IP")
				// TODO: if rs OK and push to chan
				ticker.Stop()
				logrus.WithFields(logrus.Fields{
					"res": res,
					"req": req,
				}).Info("RPC - Client received data")
				devChan <- tun.Device{
					Name:      req.Name,
					Ip:        res.Ip,
					Mtu:       res.Mtu,
					Subnets:   res.Subnets,
					ServerVIP: res.ServerVIP,
				}
			}
		}()
	}
	// END: DHCP
	go p2p.FindPeerIdsViaDHT(host, zone)
	// TODO: find peers
	// peerIds := p2p.FindPeerIds()
	// peerIds, _ := cmd.Flags().GetString("peers")
	// logrus.WithFields(logrus.Fields{
	// 	"PEERS": peerIds,
	// }).Info("")
	// streams := p2p.NewStreams(host, zone, strings.Split(peerIds, ","))
	// BEGIN: DEBUG
	// for _, peerId := range strings.Split(peerIds, ",") {
	// 	if stream, ok := streams[peerId]; ok {
	// 		for i := 0; i < 3; i++ {
	// 			// read until new line
	// 			bytes := []byte(fmt.Sprintf("%d", i))
	// 			bytes = append(bytes, "\n"...)
	// 			stream.Write(bytes)
	// 		}
	// 	}
	// }
	if pub := p2p.NewPubSub(host, "route"); pub != nil {
		pub.Publish(host.ID().Pretty(), config.Dev.Subnets)
	}
	/*
		// BEGIN: handler route PubSub
		topic, sub := p2p.NewPubSub(host, "route")
		if topic != nil && sub != nil {
			message := &p2p.Message{
				Id:          host.ID().Pretty(),
				MessageType: p2p.MessageTypeOnline,
				Subnets:     config.Dev.Subnets,
			}
			if bytes, err := json.Marshal(message); err == nil {
				ticker := time.NewTicker(10 * time.Second)
				go func(tk *time.Ticker) {
					interval := 10
					for {
						select {
						case <-tk.C:
							if err := topic.Publish(context.Background(), bytes); err != nil {
								logrus.WithFields(logrus.Fields{
									"ERROR":   err,
									"Message": message,
								}).Error("Publish message from to topic error")
							}
							if interval < 30*60 {
								// half of hour for the longest
								interval *= 2
							}
							ticker = time.NewTicker(time.Duration(interval) * time.Second)
						}
					}
				}(ticker)
			}
			go func() {
				for {
					if msg, err := sub.Next(context.Background()); err == nil {
						message := new(p2p.Message)
						if err := json.Unmarshal(msg.Data, message); err != nil {
							logrus.WithFields(logrus.Fields{
								"ERROR": err,
							}).Error("Parse message from topic error")
						} else {
							logrus.WithFields(logrus.Fields{
								"Message": message,
							}).Info("Receive message from topic")
							if message.MessageType == p2p.MessageTypeRoute {
								// TODO: add MASQUERADE if self
								// refresh route table - delete exist route if match then add table
								tun.RefreshRoute(message.Subnets)
							}
						}
					} else {
						logrus.WithFields(logrus.Fields{
							"ERROR": err,
						}).Error("Subscribe error")
					}
				}
			}()
		}
		// END: handler route PubSub
	*/

	select {
	case dev := <-devChan:
		// BEGIN: TUN
		mainDev = dev
		tun.NewTun(dev)
		// avoid create duplicate
		close(devChan)
		for {
			var frame ethernet.Frame
			frame.Resize(int(config.Dev.Mtu))
			n, err := tun.Read([]byte(frame))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"ERROR": err,
				}).Error("Read packet from TUN error")
			}
			frame = frame[:n]
			if frame != nil && len(frame) > 0 {
				if !waterutil.IsIPv6(frame) {
					// Only process IPv4 packet
					logrus.WithFields(logrus.Fields{
						"SRC": waterutil.IPv4Source(frame).String(),
						"DST": waterutil.IPv4Destination(frame).String(),
					}).Info("TUN - Packet SRC and DST")
					if waterutil.IPv4Source(frame).String() == waterutil.IPv4Destination(frame).String() {
						// FIXME: need't check src and dst ?
					} else {
						// TODO: froward to exactlly socket
						// for id, stream := range streams {
						// 	// TODO: check id and route
						// 	if id != "" {
						// 		bytes := append(frame, "\n"...)
						// 		stream.Write(bytes)
						// 	}
						// }
						p2p.ForwardPacket(host, zone, frame)
					}
				}
			}
		}
		// END: TUN
	}
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGINT:
				// TODO: ctrl+c - 退出
				logrus.WithFields(logrus.Fields{
					"SIG": sig,
				}).Info("Exit for SIGINT")
				tun.Close(mainDev)
				os.Exit(0)
			case syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT:
				logrus.WithFields(logrus.Fields{
					"SIG": sig,
				}).Info("Receive SIGHUP/SIGTERM/SIGQUIT but ignore currently")
			default:
				logrus.WithFields(logrus.Fields{
					"SIG": sig,
				}).Info("默认信号")
			}
		}
	}()
	ch := make(chan int, 1)
	<-ch
}

func readData(stream network.Stream, rw *bufio.ReadWriter) {
	for {
		// bytes, err := rw.ReadBytes('\n')
		bytes, isPrefix, err := rw.ReadLine()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR":  err,
				"PREFIX": isPrefix,
			}).Error("READY TO READ DATA")
			return
		}
		logrus.WithFields(logrus.Fields{
			"LocalPeer":  stream.Conn().LocalPeer().Pretty(),
			"RemotePeer": stream.Conn().RemotePeer().Pretty(),
		}).Info("Read data from stream")
		// Write to TUN
		n, err := tun.Write(bytes)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
				"SIZE":  n,
			}).Error("Write to TUN error")
		}
	}
}
