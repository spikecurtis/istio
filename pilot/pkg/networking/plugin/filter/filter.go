// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filter

import (
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/networking/plugin"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	pbHTTP "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	pbGoogle "github.com/gogo/protobuf/types"
)

type Plugin struct{}

func (Plugin) OnOutboundListener(in *plugin.InputParams, mutable *plugin.MutableObjects) error {
	return nil
}

func (Plugin) OnInboundListener(in *plugin.InputParams, mutable *plugin.MutableObjects) error {
	for _, chain := range mutable.FilterChains {
		if in.ListenerType == plugin.ListenerTypeHTTP {
			httpFilters := getHttpFilters(in)
			for _, f := range httpFilters {
				f.addToChain(&chain)
			}
		}
		tcpFilters := getTCPFilters(in)
		for _, f := range tcpFilters {
			f.addToChain(chain)
		}
	}
	return nil
}

func (Plugin) OnOutboundCluster(env model.Environment, node model.Proxy, service *model.Service, servicePort *model.Port,
	cluster *v2.Cluster) {
	return
}

func (Plugin) OnInboundCluster(env model.Environment, node model.Proxy, service *model.Service, servicePort *model.Port,
	cluster *v2.Cluster) {
	return
}

func (Plugin) OnOutboundRouteConfiguration(in *plugin.InputParams, routeConfiguration *v2.RouteConfiguration) {
	return
}

func (Plugin) OnInboundRouteConfiguration(in *plugin.InputParams, routeConfiguration *v2.RouteConfiguration) {
	return
}

type httpFilterConfig struct {
	Name   string
	Config pbGoogle.Struct
}

func newHttpFilterConfig(c *model.Config) httpFilterConfig {
	fc := httpFilterConfig{}
	return fc
}

func getHttpFilters(in *plugin.InputParams) []httpFilterConfig {
	var out []httpFilterConfig
	cfgs := in.Env.IstioConfigStore.CustomFilters(in.Service.Hostname, in.Node)
	for _, c := range cfgs {
		out = append(out, newHttpFilterConfig(c))
	}
	return out
}

func (f httpFilterConfig) addToChain(chain *plugin.FilterChain) {
	filter := pbHTTP.HttpFilter{
		Name:   f.Name,
		Config: &f.Config,
	}
	chain.HTTP = append(chain.HTTP, &filter)
}
