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
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/liloew/gvn/dhcp"
	"github.com/liloew/gvn/p2p"
	"github.com/liloew/gvn/route"
	"github.com/liloew/gvn/tun"
	"github.com/sirupsen/logrus"
	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water/waterutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	upCmd = &cobra.Command{
		Use:   "up",
		Short: "run gvn",
		Long:  `Run gvn using the configure file (gvn.yaml)`,
		Run: func(cmd *cobra.Command, args []string) {
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
		go readData(stream)
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
		dhcp.NewRPCServer(host, rpcZone, viper.GetString("dev.vip"), viper.GetInt("dev.mtu"), viper.GetStringSlice("dev.subnets"))
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
			// HEARTBEAT
			req := dhcp.Request{
				Id:      viper.GetString("id"),
				Name:    viper.GetString("dev.name"),
				Subnets: viper.GetStringSlice("dev.subnets"),
			}
			if client, res := dhcp.NewRPCClient(host, rpcZone, viper.GetString("server"), req); client != nil {
				// ticker.Stop()
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
					Port:      viper.GetUint("port"),
				}
			}
			ticker := time.NewTicker(INTERVAL * time.Second)
			for range ticker.C {
				var ress []dhcp.Response
				if err := dhcp.Call("DHCPService", "Clients", req, &ress); err == nil {
					for _, r := range ress {
						logrus.WithFields(logrus.Fields{
							"VIP":    r.Ip,
							"ID":     r.Id,
							"Subnet": r.Subnets,
						}).Debug("Refresh local vip table")
						subnet := strings.Split(r.Ip, "/")[0] + "/32"
						route.EventBus.Publish(route.ADD_ROUTE_TOPIC, route.RouteEvent{Id: r.Id, Subnets: []string{subnet}})
						if r.Id != host.ID().Pretty() {
							// does not change route via local ethernet
							route.EventBus.Publish(route.ADD_ROUTE_TOPIC, route.RouteEvent{Id: r.Id, Subnets: r.Subnets})
						}
					}
				} else {
					logrus.WithFields(logrus.Fields{
						"ERROR": err,
					}).Error("Request clients error")
				}
				if err := dhcp.Call("DHCPService", "Ping", req, nil); err != nil {
					logrus.WithFields(logrus.Fields{
						"ERROR": err,
					}).Error("RPC - Ping error")
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
				// exit when receive ctrl+c and others signal
				logrus.WithFields(logrus.Fields{
					"SIG": sig,
					"dev": mainDev,
				}).Info("Exit for SIGINT")
				filename := filepath.Join(os.TempDir(), "gvn.pid")
				os.Remove(filename)
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
	/*
		vip := strings.Split(mainDev.Ip, "/")[0]
		if pub != nil {
			pub.Publish(host.ID().Pretty(), vip, config.Dev.Subnets)
		}
	*/
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

	// wirte pid file
	filename := filepath.Join(os.TempDir(), "gvn.pid")
	var pidfile *os.File
	if pidfile, err = os.Create(filename); err != nil {
		os.Remove(filename)
		pidfile, _ = os.Create(filename)
	}
	if err := ioutil.WriteFile(pidfile.Name(), []byte(fmt.Sprintf("%v", os.Getpid())), 0664); err == nil {
		logrus.WithFields(logrus.Fields{
			"PID":  os.Getpid(),
			"File": pidfile.Name(),
		}).Info("Write pid file success")
	} else {
		logrus.WithFields(logrus.Fields{
			"PID":  os.Getpid(),
			"File": pidfile.Name(),
		}).Error("Write pid file error")
	}
	pidfile.Close()

	select {}
}

func readData(stream network.Stream) {
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
			stream.Close()
			break
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
