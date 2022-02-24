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
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
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
	pub     *p2p.Publisher
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

	zone := fmt.Sprintf("/gvn/%s", viper.GetString("version"))
	rpcZone := fmt.Sprintf("/rpc/%s", viper.GetString("version"))
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

	var bootstraps []string
	if MODE(viper.GetUint("mode")) == MODECLIENT {
		// start dht in server mode
		bootstraps = viper.GetStringSlice("server")
	} else {
		// DHT connect to self
		for _, addr := range host.Addrs() {
			bootstraps = append(bootstraps, fmt.Sprintf("%s/p2p/%s", addr.String(), host.ID().Pretty()))
		}
		dhcp.NewRPCServer(host, rpcZone, viper.GetString("dev.vip"), viper.GetInt("dev.mtu"))
		// auto config in server mode
		devChan <- tun.Device{
			Name: viper.GetString("dev.name"),
			Ip:   viper.GetString("dev.vip"),
			Mtu:  viper.GetInt("dev.mtu"),
			// Subnets:   viper.GetStringSlice("dev.subnets"),
			ServerVIP: viper.GetString("dev.vip"),
			Port:      viper.GetUint("port"),
		}
	}
	p2p.NewDHT(host, zone, bootstraps)

	// DHCP for client mode
	if MODE(viper.GetUint("mode")) == MODECLIENT {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			for _ = range ticker.C {
				req := dhcp.Request{
					Id:      viper.GetString("id"),
					Name:    viper.GetString("dev.name"),
					Subnets: viper.GetStringSlice("dev.subnets"),
				}
				if client, res := dhcp.NewRPCClient(host, rpcZone, viper.GetString("server"), req); client != nil {
					ticker.Stop()
					logrus.WithFields(logrus.Fields{
						"res": res,
						"req": req,
					}).Info("RPC - Client received data")
					devChan <- tun.Device{
						Name: req.Name,
						Ip:   res.Ip,
						Mtu:  res.Mtu,
						// ignore subnets because of self did't forward it to TUN
						// Subnets:   res.Subnets,
						ServerVIP: res.ServerVIP,
					}

					// TODO: sleep or make sure call after tun.New
					// refresh local VIP table
					var ress []dhcp.Response
					if err := dhcp.Call("DHCPService", "Clients", req, &ress); err == nil {
						subnets := make([]string, 0)
						for _, r := range ress {
							logrus.WithFields(logrus.Fields{
								"VIP":    r.Ip,
								"ID":     r.Id,
								"Subnet": r.Subnets,
							}).Info("Refresh local vip table")
							p2p.RouteTable.AddByString(strings.Split(r.Ip, "/")[0]+"/32", r.Id)
							if r.Id != host.ID().Pretty() {
								subnets = append(subnets, r.Subnets...)
							}
						}
						logrus.WithFields(logrus.Fields{
							"subnets": subnets,
						}).Info("Refresh subnets")
						tun.RefreshRoute(subnets)
					}
				}
			}
		}()
	}
	// END: DHCP
	go p2p.FindPeerIdsViaDHT(host, zone)
	pub = p2p.NewPubSub(host, "route")

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGINT:
				// exit when receive ctrl+c
				logrus.WithFields(logrus.Fields{
					"SIG": sig,
					"dev": mainDev,
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
				}).Info("Default ignore current SIG")
			}
		}
	}()

	mainDev = <-devChan
	logrus.WithFields(logrus.Fields{
		"dev": mainDev,
	}).Info("Create TUN device")
	tun.NewTun(mainDev)
	// avoid create duplicate
	close(devChan)
	vip := strings.Split(mainDev.Ip, "/")[0]
	if pub != nil {
		pub.Publish(host.ID().Pretty(), vip, config.Dev.Subnets)
	}
	_, vipNet, err := net.ParseCIDR(mainDev.Ip)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Fatal("Device VIP error")
	}
	go func() {
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
					}).Debug("TUN - Packet SRC and DST")
					if waterutil.IPv4Source(frame).String() != waterutil.IPv4Destination(frame).String() {
						p2p.ForwardPacket(host, zone, frame, vipNet)
					}
				}
			}
		}
	}()

	select {}
}

func readData(stream network.Stream, rw *bufio.ReadWriter) {
	for {
		var psize = make([]byte, 2)
		if _, err := stream.Read(psize); err != nil {
			stream.Close()
		}
		size := binary.LittleEndian.Uint16(psize)
		bytes := make([]byte, size)
		n, err := stream.Read(bytes[:size])
		if err != nil || n <= 0 {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
				"SIZE":  n,
			}).Error("Read data error")
			if err.Error() == "EOF" {
				break
			}
			continue
		}
		logrus.WithFields(logrus.Fields{
			"LocalPeer":  stream.Conn().LocalPeer().Pretty(),
			"RemotePeer": stream.Conn().RemotePeer().Pretty(),
		}).Debug("Read data from stream")
		// Write to TUN
		n, err = tun.Write(bytes)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
				"SIZE":  n,
			}).Error("Write to TUN error")
		}
	}
}
