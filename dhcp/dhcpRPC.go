package dhcp

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

var (
	MaxCIDR string
	mu      sync.Mutex
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
	// bind to database or KV store
	KV map[string]Response
	// should be 192.168.1.1/24 for example
	Cidr string
}

func (s *DHCPService) DHCP(ctx context.Context, req Request, res *Response) error {
	logrus.WithFields(logrus.Fields{
		"Request": req,
	}).Info("RPC - request")
	mu.Lock()
	data, ok := s.KV[req.Id]
	if !ok {
		data.Id = req.Id
		data.Name = req.Name
		data.Subnets = req.Subnets
		// TODO: find and increase
		// res.Ip = "" + "/24"
		// s.KV[req.Id] = *res
		// Calculate ip whitin the same subnet

		if MaxCIDR == "" {
			MaxCIDR = s.Cidr
		}
		ipv4Addr, ipv4Net, err := net.ParseCIDR(MaxCIDR)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"PEER": MaxCIDR,
			}).Fatal("IP addr is invalid")
		}
		ipv4Addr = ipv4Addr.To4()
		ipv4Addr[3]++
		if ipv4Net.Contains(ipv4Addr) {
			MaxCIDR = ipv4Addr.String() + "/" + strings.Split(MaxCIDR, "/")[1]
		} else {
			// TODO: loop for an available ip
		}
		data.Ip = MaxCIDR
	}
	s.KV[req.Id] = data
	// res = &data
	res.Id = data.Id
	res.Ip = data.Ip
	res.Name = data.Name
	res.Mtu = data.Mtu
	res.Subnets = data.Subnets
	mu.Unlock()
	return nil
}

func NewRPCServer(host host.Host, zone string, cidr string) {
	server := rpc.NewServer(host, protocol.ID(zone))
	service := DHCPService{KV: map[string]Response{}, Cidr: cidr}
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
