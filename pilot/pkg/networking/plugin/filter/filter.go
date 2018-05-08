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
	"istio.io/istio/pkg/log"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	pbListener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	pbHTTP "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"istio.io/api/networking/v1alpha3"
)

type Plugin struct{}

func NewPlugin() plugin.Plugin {
	return Plugin{}
}

func (Plugin) OnOutboundListener(in *plugin.InputParams, mutable *plugin.MutableObjects) error {
	return nil
}

func (Plugin) OnInboundListener(in *plugin.InputParams, mutable *plugin.MutableObjects) error {
	filters := getFilters(in, v1alpha3.FilterAugment_INBOUND)
	log.Infof("OnInboundListener got %d filter augments", len(filters))
	for _, chain := range mutable.FilterChains {
		for _, f := range filters {
			addFilterToChain(&chain, f)
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

func getFilters(in *plugin.InputParams, d v1alpha3.FilterAugment_Direction) []*v1alpha3.FilterAugment {
	var out []*v1alpha3.FilterAugment
	var hostname model.Hostname
	if in.Service != nil {
		// Outbound listeners
		hostname = in.Service.Hostname
	} else if in.ServiceInstance != nil && in.ServiceInstance.Service != nil {
		// Inbound listeners
		hostname = in.ServiceInstance.Service.Hostname
	}
	cfgs := in.Env.IstioConfigStore.FilterAugments(hostname, in.Node)
	log.Infof("Got %d augments before checking matches", len(cfgs))
	for _, c := range cfgs {
		aug := c.Spec.(*v1alpha3.FilterAugment)
		log.Infof("Checking FilterAugment %s", aug.String())
		if filterMatches(aug, in, d) {
			out = append(out, aug)
		}
	}
	return out
}

func filterMatches(f *v1alpha3.FilterAugment, in *plugin.InputParams, d v1alpha3.FilterAugment_Direction) bool {
	return listenerTypeMatch(f.GetListenerTypes(), in.ListenerType) && f.GetDirection() == d
}

func listenerTypeMatch(lts []v1alpha3.FilterAugment_ListenerType, lt plugin.ListenerType) bool {
	for _, l := range lts {
		if convertListenerType(l) == lt {
			return true
		}
	}
	return false
}

func convertListenerType(in v1alpha3.FilterAugment_ListenerType) plugin.ListenerType {
	switch in {
	case v1alpha3.FilterAugment_HTTP:
		return plugin.ListenerTypeHTTP
	case v1alpha3.FilterAugment_TCP:
		return plugin.ListenerTypeTCP
	default:
		return plugin.ListenerTypeUnknown
	}
}

func addFilterToChain(chain *plugin.FilterChain, f *v1alpha3.FilterAugment) {
	switch f.GetFilter().(type) {
	case *v1alpha3.FilterAugment_HttpFilter:
		addHTTPFilterToChain(chain, f)
	case *v1alpha3.FilterAugment_NetworkFilter:
		addTCPFilterToChain(chain, f)
	}
}

func addHTTPFilterToChain(chain *plugin.FilterChain, aug *v1alpha3.FilterAugment) {
	f := aug.GetHttpFilter()
	filter := pbHTTP.HttpFilter{
		Name:   f.Name,
		Config: f.Config,
	}
	var names []string
	for _, j := range chain.HTTP {
		names = append(names, j.Name)
	}
	i := insertIndex(names, aug)

	// Insert into new slice.
	oldFilters := chain.HTTP
	chain.HTTP = make([]*pbHTTP.HttpFilter, len(oldFilters)+1)
	for n := 0; n < i; n++ {
		chain.HTTP[n] = oldFilters[n]
	}
	chain.HTTP[i] = &filter
	for n := i; n < len(oldFilters); n++ {
		chain.HTTP[n+1] = oldFilters[n]
	}
}

func addTCPFilterToChain(chain *plugin.FilterChain, aug *v1alpha3.FilterAugment) {
	f := aug.GetNetworkFilter()
	filter := pbListener.Filter{
		Name:   f.Name,
		Config: f.Config,
	}
	var names []string
	for _, j := range chain.TCP {
		names = append(names, j.Name)
	}
	i := insertIndex(names, aug)

	// Insert into new slice.
	oldFilters := chain.TCP
	chain.TCP = make([]pbListener.Filter, len(oldFilters)+1)
	for n := 0; n < i; n++ {
		chain.TCP[n] = oldFilters[n]
	}
	chain.TCP[i] = filter
	for n := i; n < len(oldFilters); n++ {
		chain.TCP[n+1] = oldFilters[n]
	}
}

func insertIndex(names []string, aug *v1alpha3.FilterAugment) int {
	switch aug.GetOrder().GetPosition() {
	case v1alpha3.FilterAugment_Order_FIRST:
		return 0
	case v1alpha3.FilterAugment_Order_LAST:
		return len(names)
	case v1alpha3.FilterAugment_Order_BEFORE:
		target := aug.GetOrder().GetRelativeTo()
		for i, n := range names {
			if n == target {
				return i
			}
		}
		return len(names)
	case v1alpha3.FilterAugment_Order_AFTER:
		target := aug.GetOrder().GetRelativeTo()
		for i, n := range names {
			if n == target {
				return i + 1
			}
		}
		return len(names)
	}
	panic("unknown Position")
}
