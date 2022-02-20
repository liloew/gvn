package p2p

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/sirupsen/logrus"
)

type MessageType uint
type MessageHandler pubsub.TopicEventHandlerOpt

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
	// TODO:
	// defer host.Close()
	logrus.WithFields(logrus.Fields{
		"Addrs": host.Addrs(),
		"ID":    host.ID(),
		"PEERS": host.Peerstore().Peers(),
	}).Debug("create Peer successful")
	return host, nil
}

func NewPubSub(host host.Host, topic string) (*pubsub.Topic, *pubsub.Subscription) {
	// TODO: host has join DHT
	ctx, _ := context.WithCancel(context.Background())
	// defer cancle()
	// BEGIN: DEBUG
	tmpfile, err := ioutil.TempFile("", "PubSub-Tracer-*.json")
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"File":  tmpfile,
			"ERROR": err,
		}).Error("Creat tmp file error")
	}
	tracer, err := pubsub.NewJSONTracer(tmpfile.Name())
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Creat tracer error")
	}
	logrus.WithFields(logrus.Fields{
		"File": tmpfile.Name(),
	}).Info("Trace file")
	// END: DEBUG
	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithEventTracer(tracer), pubsub.WithDiscovery(routingDiscovery))
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
		return tc, sub
	}
	return nil, nil
}

func ForwardPacket(packets []byte) {
}
