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
package p2p

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/liloew/gvn/dhcp"
	"github.com/liloew/gvn/route"
	"github.com/liloew/gvn/tun"
	"github.com/sirupsen/logrus"
	"github.com/songgao/water/waterutil"
	"github.com/spf13/viper"
)

type MessageType uint
type MessageHandler pubsub.TopicEventHandlerOpt

type Publisher struct {
	pub *pubsub.Topic
	sub *pubsub.Subscription
}

type Publish interface {
	Publish(subnets []string)
}

const (
	MessageTypeOnline MessageType = iota
	MessageTypeOffline
	MessageTypeRoute
)

type Message struct {
	Id          string      `json:"id"`
	MessageType MessageType `json:"messageType"`
	Vip         string      `json:"vip"`
	Subnets     []string    `json:subnets`
}

var (
	Streams = make(map[string]network.Stream)
	VIP     string
)

func init() {

}

func NewPeer(priKey string, port uint) (host.Host, error) {
	pk, err := crypto.UnmarshalPrivateKey([]byte(priKey))
	if err != nil {
		return nil, err
	}
	host, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(pk),
		libp2p.ForceReachabilityPublic(),
		libp2p.FallbackDefaults,
	)
	if err != nil {
		return host, err
	}
	// defer host.Close()
	logrus.WithFields(logrus.Fields{
		"Addrs": host.Addrs(),
		"ID":    host.ID(),
		"PEERS": host.Peerstore().Peers(),
	}).Debug("create Peer successful")
	return host, nil
}

func NewPubSub(host host.Host, topic string) *Publisher {
	ctx, _ := context.WithCancel(context.Background())
	// defer cancle()
	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithDiscovery(routingDiscovery))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("Create PubSub error")
	}
	if tc, err := ps.Join(topic); err != nil {
		// TODO: re-join
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Create PubSub error")
	} else {
		sub, err := tc.Subscribe()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
			}).Error("Subscribe error")
		}

		go func() {
			for {
				if msg, err := sub.Next(context.Background()); err == nil {
					message := new(Message)
					if err := json.Unmarshal(msg.Data, message); err != nil {
						logrus.WithFields(logrus.Fields{
							"ERROR": err,
						}).Error("Parse message from topic error")
					} else {
						logrus.WithFields(logrus.Fields{
							"Message": message,
						}).Info("Receive message from topic")
						if message.MessageType == MessageTypeRoute {
							if msg.ReceivedFrom.Pretty() == host.ID().Pretty() {
								continue
							}
							// TODO: add MASQUERADE if self
							tun.RefreshRoute(message.Subnets)
							for _, subnet := range message.Subnets {
								// Add will override the exist one
								route.Route.Add(strings.TrimSpace(subnet), message.Id)
							}
						} else if message.MessageType == MessageTypeOnline {
							// refresh clients
							if viper.GetUint("mode") == 1 {
								// server
								continue
							} else {
								req := dhcp.Request{}
								var ress []dhcp.Response
								if err := dhcp.Call("DHCPService", "Clients", req, &ress); err == nil {
									subnets := make([]string, 0)
									for _, r := range ress {
										if r.Id == host.ID().Pretty() {
											continue
										}
										logrus.WithFields(logrus.Fields{
											"VIP":    r.Ip,
											"ID":     r.Id,
											"Subnet": r.Subnets,
										}).Info("Refresh local vip table")
										route.Route.Add(strings.Split(r.Ip, "/")[0]+"/32", r.Id)
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
						} else if message.MessageType == MessageTypeOffline {
							route.Route.Remove(strings.Split(message.Vip, "/")[0] + "/32")
						}
					}
				} else {
					logrus.WithFields(logrus.Fields{
						"ERROR": err,
					}).Error("Subscribe error")
				}
			}
		}()

		return &Publisher{
			pub: tc,
			sub: sub,
		}
	}
	return nil
}

func (p *Publisher) Publish(peerId string, vip string, subnets []string) {
	message := &Message{
		Id:          peerId,
		MessageType: MessageTypeRoute,
		Vip:         vip,
	}
	if len(subnets) > 0 {
		message.Subnets = subnets
	} else {
		message.MessageType = MessageTypeOnline
	}
	if bytes, err := json.Marshal(message); err == nil {
		ticker := time.NewTicker(10 * time.Second)
		go func(tk *time.Ticker) {
			interval := 10
			for _ = range tk.C {
				if err := p.pub.Publish(context.Background(), bytes); err != nil {
					logrus.WithFields(logrus.Fields{
						"ERROR":   err,
						"Message": message,
					}).Error("Publish message from to topic error")
				}
				if interval < 30*60 {
					// half of hour for the longest
					interval *= 2
				}
				logrus.WithFields(logrus.Fields{
					"Interval": interval,
				}).Info("")
				ticker.Reset(time.Duration(interval) * time.Second)
			}
		}(ticker)
	}
}

func ForwardPacket(host host.Host, zone string, packets []byte, vipNet *net.IPNet) {
	dst := waterutil.IPv4Destination(packets)
	if peerId, found, err := route.Route.Get(dst.String()); err == nil && found {
		if stream, ok := Streams[peerId.(string)]; ok {
			binary.Write(stream, binary.LittleEndian, uint16(len(packets)))
			if n, err := stream.Write(packets); n != len(packets) || err != nil {
				logrus.WithFields(logrus.Fields{
					"ERROR": err,
					"SIZE":  n,
				}).Error("Forward to stream error")
				if err.Error() == "stream reset" {
					stream.Close()
					delete(Streams, peerId.(string))
				}
			}
		} else {
			// make new stream
			if s := NewStream(host, zone, peerId.(string)); s != nil {
				Streams[peerId.(string)] = s
				ForwardPacket(host, zone, packets, vipNet)
			} else {
				logrus.WithFields(logrus.Fields{
					"ERROR": err,
				}).Error("Forward to stream error")
			}
		}
	} else {
		// discard
		logrus.WithFields(logrus.Fields{
			"SRC": waterutil.IPv4Source(packets).String(),
			"DST": dst,
		}).Error("Discard")
	}
}
