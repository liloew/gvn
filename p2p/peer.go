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
	"github.com/liloew/altgvn/tun"
	"github.com/sirupsen/logrus"
	"github.com/songgao/water/waterutil"
	"github.com/zmap/go-iptree/iptree"
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
	RouteTable = iptree.New()
	Streams    = make(map[string]network.Stream)
	VIP        string
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
					if msg.ReceivedFrom.Pretty() == host.ID().Pretty() {
						continue
					}
					if err := json.Unmarshal(msg.Data, message); err != nil {
						logrus.WithFields(logrus.Fields{
							"ERROR": err,
						}).Error("Parse message from topic error")
					} else {
						logrus.WithFields(logrus.Fields{
							"Message": message,
						}).Info("Receive message from topic")
						if message.MessageType == MessageTypeRoute {
							// TODO: add MASQUERADE if self
							tun.RefreshRoute(message.Subnets)
							for _, subnet := range message.Subnets {
								// Add will override the exist one
								RouteTable.AddByString(strings.TrimSpace(subnet), message.Id)
							}
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
	if len(subnets) > 0 {
		message := &Message{
			Id:          peerId,
			MessageType: MessageTypeRoute,
			Vip:         vip,
			Subnets:     subnets,
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
}

func ForwardPacket(host host.Host, zone string, packets []byte, vipNet *net.IPNet) {
	dst := waterutil.IPv4Destination(packets)
	if peerId, found, err := RouteTable.GetByString(dst.String()); err == nil && found {
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
