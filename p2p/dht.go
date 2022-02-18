package p2p

import (
	"context"
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

// func NewDHT(host host.Host, bootstraps []multiaddr.Multiaddr) *dht.IpfsDHT {
func NewDHT(host host.Host, zone string, bootstraps []string) *dht.IpfsDHT {
	ctx := context.Background()
	addrs := make([]peer.AddrInfo, 0)
	if len(bootstraps) > 0 {
		for _, bootstrap := range bootstraps {
			// ids := strings.Split(bootstrap, "/")
			// bs := peer.AddrInfo{ID: peer.ID(ids[len(ids)-1]), Addrs: addrs}
			addr, err := peer.AddrInfoFromString(bootstrap)
			if err != nil {
				continue
			}
			addrs = append(addrs, *addr)
		}

		// var options []dht.Option
		// options = append(options, dht.Mode(dht.ModeServer))
		// options = append(options, dht.BootstrapPeers(bs))

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
func FindPeerIdsViaPubSub() []string {
	return nil
}

func FindPeerIdsViaDHT(host host.Host, zone string) []string {
	// TODO: check kdht nil
	// TODO: multiplex the connection
	peerIds := make([]string, 0)
	/*
		peers := kdht.Host().Network().Peers()
		for _, peer := range peers {
			peerIds = append(peerIds, peer.Pretty())
		}
	*/
	// BEGIN: DEBUG
	ticker := time.NewTicker(20 * time.Second)
	go func(tk *time.Ticker) {
		for {
			select {
			case <-tk.C:
				// TODO:
				peers, err := routingDiscovery.FindPeers(context.Background(), zone)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"ERROR": err,
					}).Error("Error at timer")
					continue
				}
				for peer := range peers {
					logrus.WithFields(logrus.Fields{
						"PEER": peer.ID.Pretty(),
					}).Info("Peers at timer")
				}
			}
		}
	}(ticker)
	// END: DEBUG
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
	/*
		if host.Network().Connectedness(peerId) != network.Connected {
			addrs, err := kdht.FindPeer(ctx, peerId)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"ID":    id,
					"addrs": addrs,
				}).Error("unable match peer")
				continue
			}
			stream, err := host.NewStream(ctx, addrs.ID, protocol.ID(zone))
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"ID":       id,
					"addrs":    addrs,
					"protocol": zone,
				}).Error("new stream to peer error")
				continue
			}
			streams[id] = stream
		} else if _, ok := streams[id]; !ok {
			adrs1 := host.Network().Peerstore().Addrs(peerId)
			logrus.WithFields(logrus.Fields{
				"ID":       id,
				"addrs1":   adrs1,
				"protocol": zone,
			}).Error("new stream to peer error")
			for _, addrs := range adrs1 {
				adr, err := multiaddr.NewMultiaddr(fmt.Sprintf("%s%s%s", addrs.String(), protocol.ID(zone), id))
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"ID":       id,
						"addrs":    addrs,
						"protocol": zone,
						"ERROR":    err,
						"ADDR":     adr,
					}).Error("Addr error")
				}
				stream, err := host.NewStream(ctx, peerId, protocol.ID(zone))
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"ID":       id,
						"addrs":    addrs,
						"protocol": zone,
						"ERROR":    err,
					}).Error("new stream to peer error")
					continue
				}
				streams[id] = stream
				break
			}
		}
	*/
}
