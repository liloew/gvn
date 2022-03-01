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
package dhcp

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/liloew/gvn/route"
	"github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

var (
	ServerVIP string
	MaxCIDR   string
	Mtu       int
	mu        sync.Mutex
	client    *rpc.Client
	serverId  peer.ID
)

type RPC struct {
}

type Request struct {
	Id      string
	Mode    int
	Name    string
	Subnets []string
}

type Response struct {
	Id        string
	Name      string
	Ip        string
	Mtu       int
	Subnets   []string
	ServerVIP string
	Mode      int
	Ttl       int64
	LoginTime int64
}

type DHCPService struct {
	// bind to database or KV store
	KV map[string]Response
	// should be 192.168.1.1/24 for example
	Cidr string
	Mtu  int
}

func (s *DHCPService) DHCP(ctx context.Context, req Request, res *Response) error {
	logrus.WithFields(logrus.Fields{
		"Request": req,
	}).Info("RPC call Clients")
	mu.Lock()
	data, ok := s.KV[req.Id]
	if !ok {
		data.Id = req.Id
		data.Name = req.Name
		data.Subnets = req.Subnets
		data.Mode = req.Mode
		data.LoginTime = time.Now().Unix()
		data.Ttl = 10 * 60 // 10 min

		if MaxCIDR == "" {
			MaxCIDR = s.Cidr
			ServerVIP = s.Cidr
			Mtu = s.Mtu
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
		data.Mtu = s.Mtu
		data.ServerVIP = ServerVIP
	}
	s.KV[req.Id] = data
	// res = &data
	res.Id = data.Id
	res.Ip = data.Ip
	res.Name = data.Name
	res.Mtu = data.Mtu
	res.Subnets = data.Subnets
	res.ServerVIP = data.ServerVIP
	res.Mode = data.Mode
	res.LoginTime = data.LoginTime
	res.Ttl = data.Ttl
	logrus.WithFields(logrus.Fields{
		"res": res,
	}).Info("RPC - Client requested data")

	// vip/mask -> vip/32
	// route.Route.Add(strings.Split(data.Ip, "/")[0]+"/32", data.Id)
	subnet := strings.Split(data.Ip, "/")[0] + "/32"
	route.EventBus.Publish(route.ADD_ROUTE_TOPIC, route.RouteEvent{Id: data.Id, Subnets: []string{subnet}})
	mu.Unlock()
	return nil
}

func (s *DHCPService) Clients(ctx context.Context, req Request, res *[]Response) error {
	mu.Lock()
	for _, v := range s.KV {
		if v.Mode != 1 && time.Now().Unix()-v.LoginTime > v.Ttl {
			continue
		}
		r := Response{
			Id:        v.Id,
			Name:      v.Name,
			Ip:        v.Ip,
			Mtu:       v.Mtu,
			Mode:      v.Mode,
			Subnets:   v.Subnets,
			ServerVIP: v.ServerVIP,
		}
		*res = append(*res, r)
	}
	mu.Unlock()
	return nil
}

func (s *DHCPService) Ping(ctx context.Context, req Request, res *Response) error {
	mu.Lock()
	if v, ok := s.KV[req.Id]; ok {
		v.Ttl = 10 * 60 // 10 min
		v.LoginTime = time.Now().Unix()
		s.KV[req.Id] = v
	} else {
		mu.Unlock()
		return errors.New("not found")
	}
	mu.Unlock()
	return nil
}

func NewRPCServer(host host.Host, zone string, cidr string, mtu int) {
	server := rpc.NewServer(host, protocol.ID(zone))
	service := DHCPService{KV: map[string]Response{}, Cidr: cidr, Mtu: mtu}
	if err := server.Register(&service); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Panic("RPC - build RPC service error")
	}
	// does not clean the zombie client
	/*
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			for range ticker.C {
				for k, v := range service.KV {
					if time.Now().Unix()-v.LoginTime > v.Ttl {
						mu.Lock()
						delete(service.KV, k)
						mu.Unlock()
					}
				}
			}
		}()
	*/
	// server register
	mu.Lock()
	service.KV[host.ID().Pretty()] = Response{
		Id:        host.ID().Pretty(),
		Ip:        cidr,
		Mtu:       mtu,
		ServerVIP: cidr,
		Mode:      1,
		LoginTime: time.Now().Unix(),
		Ttl:       10 * 60, // 10 min
	}
	mu.Unlock()
}

func NewRPCClient(host host.Host, zone string, server string, req Request) (*rpc.Client, Response) {
	client = rpc.NewClient(host, protocol.ID(zone))
	var res Response
	if ma, err := multiaddr.NewMultiaddr(server); err == nil {
		if addr, err := peer.AddrInfoFromP2pAddr(ma); err == nil {
			serverId = addr.ID
			if err := client.Call(addr.ID, "DHCPService", "DHCP", req, &res); err != nil {
				logrus.WithFields(logrus.Fields{
					"ERROR": err,
				}).Error("RPC - call DHCP RPC serveice error")
			}
			return client, res
		}
	}
	return nil, res
}

func Call(svcName string, svcMethod string, req Request, res interface{}) error {
	if err := client.Call(serverId, svcName, svcMethod, req, &res); err != nil {
		logrus.WithFields(logrus.Fields{
			"ERROR":     err,
			"svcName":   svcName,
			"svcMethod": svcMethod,
			"req":       req,
			"res":       res,
		}).Error("RPC - call RPC serveice error")
		return err
	}
	return nil
}
