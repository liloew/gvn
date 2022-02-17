package p2p

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/sirupsen/logrus"
)

func NewPeer(priKey string) (host.Host, error) {
	pk, err := crypto.UnmarshalPrivateKey([]byte(priKey))
	if err != nil {
		return nil, err
	}
	host, err := libp2p.New(
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
