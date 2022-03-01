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
		offlineCh := make(chan eventbus.DataEvent)
		EventBus.Subscribe(ADD_ROUTE_TOPIC, addRouteCh)
		EventBus.Subscribe(REMOVE_ROUTE_TOPIC, removeRouteCh)
		EventBus.Subscribe(REFRESH_ROUTE_TOPIC, refreshRouteCh)
		EventBus.Subscribe(ONLINE_TOPIC, onlineCh)
		EventBus.Subscribe(OFFLINE_TOPIC, offlineCh)
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
				// for _, subnet := range data.Data.(RouteEvent).Subnets {
				// 	Route.remove(subnet)
				// 	Route.add(subnet, data.Data.(RouteEvent).Id)
				// }
				Route.refresh(data.Data.(RouteEvent).Subnets, data.Data.(RouteEvent).Id)
			case data := <-onlineCh:
				logrus.WithFields(logrus.Fields{
					"Data":  data.Data,
					"Topic": data.Topic,
				}).Debug("Online Channel")
			case data := <-offlineCh:
				logrus.WithFields(logrus.Fields{
					"Data":  data.Data,
					"Topic": data.Topic,
				}).Debug("Offline Channel")
			}
		}
	}()
}

type RouteTable struct {
	tree *iptree.IPTree
	rm   sync.RWMutex
}

func (r *RouteTable) refresh(subnets []string, peerId string) {
	removed := oneSideSlice(LocalRoute, subnets)
	added := oneSideSlice(subnets, LocalRoute)
	logrus.WithFields(logrus.Fields{
		"Removed":    removed,
		"Added":      added,
		"LocalRoute": LocalRoute,
		"Subnets":    subnets,
	}).Debug("Remove and Added subnets")
	for _, v := range removed {
		r.remove(v)
	}
	// for _, v := range mergeSlice(LocalRoute, subnets) {
	for _, v := range added {
		r.add(v, peerId)
	}
}

func (r *RouteTable) remove(subnet string) {
	r.rm.Lock()
	if err := tun.RemoveRoute([]string{subnet}); err == nil {
		r.tree.DeleteByString(subnet)
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
		}).Debug("Ignore the subnet becuase of existence")
		r.rm.Unlock()
		return
	}
	if err := tun.AddRoute([]string{subnet}); err == nil {
		r.tree.AddByString(subnet, peerId)
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

////////////////////////////////////////////////////////////////////////////////

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

// return the items co-exists in left and right
func mergeSlice(left, right []string) []string {
	tmp := make([]string, 0)
	for _, v := range left {
		if contains(right, v) && !contains(tmp, v) {
			tmp = append(tmp, v)
		}
	}
	for _, v := range right {
		if contains(left, v) && !contains(tmp, v) {
			tmp = append(tmp, v)
		}
	}
	return tmp
}

// return the items only exists left not exists right
// left:  [1, 3, 5, 7, 9]
// right: [1, 2, 3, 4]
// result: [5, 7, 9]
func oneSideSlice(left, right []string) []string {
	tmp := make([]string, 0)
	for _, v := range left {
		if !contains(right, v) {
			tmp = append(tmp, v)
		}
	}
	return tmp
}

////////////////////////////////////////////////////////////////////////////////
