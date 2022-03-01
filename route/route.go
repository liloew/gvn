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
package route

import (
	"sync"

	"github.com/liloew/gvn/eventbus"
	"github.com/liloew/gvn/tun"
	"github.com/sirupsen/logrus"
	"github.com/zmap/go-iptree/iptree"
)

type RouteEvent struct {
	Subnets []string
	Id      string
	Vip     string
}

const (
	ADD_ROUTE_TOPIC     = "ADD_ROUTE"
	REMOVE_ROUTE_TOPIC  = "REMOVE_ROUTE"
	REFRESH_ROUTE_TOPIC = "REFRESH_ROUTE"
	ONLINE_TOPIC        = "ONLINE"
	OFFLINE_TOPIC       = "OFFLINE"
)

var (
	Route = RouteTable{
		tree: iptree.New(),
	}
	EventBus = &eventbus.EventBus{
		Subscribers: map[string]eventbus.DataChannelSlice{},
	}
	LocalRoute = make([]string, 0)
)

func init() {
	go func() {
		addRouteCh := make(chan eventbus.DataEvent)
		removeRouteCh := make(chan eventbus.DataEvent)
		refreshRouteCh := make(chan eventbus.DataEvent)
		onlineCh := make(chan eventbus.DataEvent)
		// TODO: ofline channel
		EventBus.Subscribe(ADD_ROUTE_TOPIC, addRouteCh)
		EventBus.Subscribe(REMOVE_ROUTE_TOPIC, removeRouteCh)
		EventBus.Subscribe(REFRESH_ROUTE_TOPIC, refreshRouteCh)
		EventBus.Subscribe(ONLINE_TOPIC, onlineCh)
		for {
			select {
			case data := <-addRouteCh:
				logrus.WithFields(logrus.Fields{
					"Data":  data.Data,
					"Topic": data.Topic,
				}).Debug("Add Route Channel")
				for _, subnet := range data.Data.(RouteEvent).Subnets {
					Route.add(subnet, data.Data.(RouteEvent).Id)
				}
			case data := <-removeRouteCh:
				logrus.WithFields(logrus.Fields{
					"Data":  data.Data,
					"Topic": data.Topic,
				}).Debug("Remove Route Channel")
				for _, subnet := range data.Data.(RouteEvent).Subnets {
					Route.remove(subnet)
				}
			case data := <-refreshRouteCh:
				logrus.WithFields(logrus.Fields{
					"Data":  data.Data,
					"Topic": data.Topic,
				}).Debug("Refresh Route Channel")
				for _, subnet := range data.Data.(RouteEvent).Subnets {
					Route.remove(subnet)
					Route.add(subnet, data.Data.(RouteEvent).Id)
				}
			case data := <-onlineCh:
				logrus.WithFields(logrus.Fields{
					"Data":  data.Data,
					"Topic": data.Topic,
				}).Debug("Online Channel")
			}
		}
	}()
}

type RouteTable struct {
	tree *iptree.IPTree
	rm   sync.RWMutex
}

// subnet - 192.168.1.0/24
// peerId - P2P node ID
func (r *RouteTable) refresh(subnet string, peerId string) {
	r.remove(subnet)
	r.add(subnet, peerId)
}

func (r *RouteTable) remove(subnet string) {
	// TODO: LocalRoute remove
	r.rm.Lock()
	r.tree.DeleteByString(subnet)
	if err := tun.RemoveRoute([]string{subnet}); err == nil {
		LocalRoute = remove(LocalRoute, subnet)
	} else {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Remove route error")
	}
	r.rm.Unlock()
}

func (r *RouteTable) add(subnet string, peerId string) {
	r.rm.Lock()
	if contains(LocalRoute, subnet) {
		logrus.WithFields(logrus.Fields{
			"LocalRoute": LocalRoute,
			"Subnet":     subnet,
		}).Info("Ignore the subnet becuase of existence")
		r.rm.Unlock()
		return
	}
	r.tree.AddByString(subnet, peerId)
	if err := tun.AddRoute([]string{subnet}); err == nil {
		LocalRoute = append(LocalRoute, subnet)
	} else {
		logrus.WithFields(logrus.Fields{
			"ERROR": err,
		}).Error("Add route error")
	}
	r.rm.Unlock()
}

func (r *RouteTable) Get(ip string) (interface{}, bool, error) {
	return r.tree.GetByString(ip)
}

func (r *RouteTable) Clean() {
	// TODO:
	r.tree = iptree.New()
}

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func remove(items []string, item string) []string {
	newitems := []string{}
	for _, i := range items {
		if i != item {
			newitems = append(newitems, i)
		}
	}
	return newitems
}
