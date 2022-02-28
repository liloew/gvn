/*
Copyright © 2022 liluo <luolee.me@gmail.com>

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
	"github.com/zmap/go-iptree/iptree"
)

var (
	Route = RouteTable{
		tree: iptree.New(),
	}
)

type RouteTable struct {
	tree *iptree.IPTree
}

// subnet - 192.168.1.0/24
// peerId - P2P node ID
func (r *RouteTable) Refresh(subnet string, peerId string) {
	r.Remove(subnet)
	r.Add(subnet, peerId)
}

func (r *RouteTable) Remove(subnet string) {
	r.tree.DeleteByString(subnet)
}

func (r *RouteTable) Add(subnet string, peerId string) {
	r.tree.AddByString(subnet, peerId)
}

func (r *RouteTable) Get(ip string) (interface{}, bool, error) {
	return r.tree.GetByString(ip)
}

func (r *RouteTable) Clean() {
	// TODO:
	r.tree = iptree.New()
}
