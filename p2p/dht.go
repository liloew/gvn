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
	"context"
	"encoding/binary"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/sirupsen/logrus"
)

var (
	kdht             *dht.IpfsDHT
	routingDiscovery *discovery.RoutingDiscovery
)

func NewDHT(host host.Host, zone string, bootstraps []string) *dht.IpfsDHT {
	ctx := context.Background()
	addrs := make([]peer.AddrInfo, 0)
	if len(bootstraps) > 0 {
		for _, bootstrap := range bootstraps {
			addr, err := peer.AddrInfoFromString(bootstrap)
			if err != nil {
				continue
			}
			addrs = append(addrs, *addr)
		}

		dstore := dsync.MutexWrap(ds.NewMapDatastore())

		kdht = dht.NewDHT(ctx, host, dstore)
		if err := kdht.Bootstrap(ctx); err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
			}).Error("bootstrap DHT node error")
		} else {
			logrus.WithFields(logrus.Fields{}).Debug("bootstrap DHT success")
		}
		// connect to bootstrap peer
		for _, boot := range addrs {
			logrus.WithFields(logrus.Fields{
				"Bootstrap": boot.String(),
			}).Info("尝试连接节点")
			if err := host.Connect(ctx, boot); err != nil {
				logrus.WithFields(logrus.Fields{
					"Bootstrap": boot.String(),
					"ERROR":     err,
				}).Info("接 bootstrap 节点失败")
			} else {
				logrus.WithFields(logrus.Fields{
					"Bootstrap": boot.String(),
				}).Info("已连接到节点")
			}
		}
	}
	routingDiscovery = discovery.NewRoutingDiscovery(kdht)
	discovery.Advertise(ctx, routingDiscovery, zone)
	go FindPeerIdsViaDHT(host, zone)
	dht.RoutingTableRefreshPeriod(60 * time.Second)
	return kdht
}

func NewStreams(host host.Host, zone string, peerIds []string) map[string]network.Stream {
	// TODO: streams := make(map[string][]network.Stream)
	ctx, _ := context.WithCancel(context.Background())
	// defer cancel()
	streams := make(map[string]network.Stream)
	conns := host.Network().Conns()
	for _, conn := range conns {
		if _, ok := streams[conn.RemotePeer().Pretty()]; !ok {
			var stream network.Stream
			for _, ss := range conn.GetStreams() {
				if ss.Protocol() == protocol.ID(zone) {
					stream = ss
					break
				}
			}
			if stream == nil {
				ss, err := host.NewStream(ctx, conn.RemotePeer(), protocol.ID(zone))
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"ERROR":      err,
						"RemotePeer": conn.RemotePeer().Pretty(),
						"LocalPeer":  conn.LocalPeer().Pretty(),
					}).Error("Create stream err")
				} else {
					stream = ss
				}
				tpk := []byte("a")
				binary.Write(ss, binary.LittleEndian, uint16(len(tpk)))
				n, err := ss.Write(tpk)
				logrus.WithFields(logrus.Fields{
					"Size":  n,
					"ERROR": err,
				}).Info("first write state")
			}
			if stream != nil {
				streams[conn.RemotePeer().Pretty()] = stream
			}
		}
	}
	for _, id := range peerIds {
		peerId, err := peer.Decode(id)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"ERROR": err,
				"ID":    id,
			}).Error("parse ID error")
			continue
		}
		if peerId == host.ID() {
			// ignore self
			continue
		}
		if _, ok := streams[id]; !ok {
			stream, err := host.NewStream(ctx, peerId, protocol.ID(zone))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"ID":       id,
					"protocol": zone,
					"ERROR":    err,
				}).Error("new stream to peer error")
				continue
			}
			streams[id] = stream
		}

	}
	return streams
}

func NewStream(host host.Host, zone string, peerId string) network.Stream {
	ctx, _ := context.WithCancel(context.Background())
	// defer cancel()
	if id, err := peer.Decode(peerId); err == nil {
		stream, err := host.NewStream(ctx, id, protocol.ID(zone))
		if err != nil {
			return nil
		}
		return stream
	}
	return nil
}

func FindPeerIdsViaPubSub() []string {
	return nil
}

func FindPeerIdsViaDHT(host host.Host, zone string) []string {
	// TODO: check kdht nil
	// TODO: multiplex the connection
	peerIds := make([]string, 0)
	if routingDiscovery != nil {
		peers, err := routingDiscovery.FindPeers(context.Background(), zone)
		if err != nil {
			return peerIds
		}
		for peer := range peers {
			if peer.ID == host.ID() {
				continue
			}
			peerIds = append(peerIds, peer.ID.Pretty())
		}
		return peerIds
	}
	return peerIds
}
