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
package p2p

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/liloew/gvn/route"
	"github.com/sirupsen/logrus"
	"github.com/songgao/water/waterutil"
)

type MessageType uint

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
