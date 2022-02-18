package dhcp

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

type Request struct {
	Id      string
	Name    string
	Subnets []string
}

type Response struct {
	Id      string
	Name    string
	Ip      string
	Mtu     int
	Subnets []string
}

type DHCPService struct {
	// TODO: bind to database or KV store
	Data interface{}
}

func (s *DHCPService) DHCP(ctx context.Context, req Request, res *Response) error {
	logrus.WithFields(logrus.Fields{
		"Request": req,
	}).Info("RPC - request")
	res.Id = req.Id
	res.Name = req.Name
	res.Subnets = req.Subnets
	// TODO: find and increase
	res.Id = "" + "/24"
	return nil
}

func NewRPCServer(host host.Host, zone string) {
	server := rpc.NewServer(host, protocol.ID(zone))
	service := DHCPService{}
	if err := server.Register(&service); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("RPC - build RPC service error")
	}
}

func NewRPCClient(host host.Host, zone string, server string, req Request) Response {
	client := rpc.NewClient(host, protocol.ID(zone))
	var res Response
	if ma, err := multiaddr.NewMultiaddr(server); err == nil {
		if addr, err := peer.AddrInfoFromP2pAddr(ma); err == nil {
			if err := client.Call(addr.ID, "DHCPService", "DHCP", req, &res); err != nil {
				logrus.WithFields(logrus.Fields{
					"ERROR": err,
				}).Error("RPC - call RPC serveice error")
			}
			return res
		}
	}
	return res
}
