// Copyright 2018 Istio Authors.
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

package v1

import (
	"strings"

	meshconfig "istio.io/api/mesh/v1alpha1"
	routing "istio.io/api/routing/v1alpha2"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/log"
)

func buildGatewayListeners(mesh *meshconfig.MeshConfig,
	instancesOnThisNode []*model.ServiceInstance,
	allServicesInTheMesh []*model.Service,
	config model.IstioConfigStore, node model.Node) Listeners {

	gateways, err := config.List(model.Gateway.Type, model.NamespaceAll)
	if err != nil {
		log.Warnf("error listing gateways: %v", err)
		return Listeners{}
	} else if len(gateways) == 0 {
		log.Debugf("listing gateways in namespace %q, got none", model.NamespaceAll)
		return Listeners{}
	}

	targetGateways := map[string]bool{}
	for _, spec := range gateways {
		targetGateways[spec.Name] = true
	}

	// TODO: is this still right??
	gateway := gateways[0].Spec.(*routing.Gateway)
	for _, spec := range gateways[1:] {
		model.MergeGateways(gateway, spec.Spec.(*routing.Gateway))
	}

	listeners := make(Listeners, 0, len(gateway.Servers))
	for _, server := range gateway.Servers {
		// TODO: TCP

		// build physical listener
		physicalListener := buildPhysicalGatewayListener(mesh, node, instancesOnThisNode, allServicesInTheMesh, config, server)
		if physicalListener == nil {
			continue // TODO: add support for all protocols
		}

		listeners = append(listeners, physicalListener)
	}

	return listeners.normalize()
}

func buildPhysicalGatewayListener(
	mesh *meshconfig.MeshConfig,
	node model.Node,
	instancesOnThisNode []*model.ServiceInstance,
	allServicesInTheMesh []*model.Service,
	config model.IstioConfigStore,
	server *routing.Server,
) *Listener {

	opts := buildHTTPListenerOpts{
		mesh:             mesh,
		node:             node,
		instances:        instancesOnThisNode,
		routeConfig:      nil,
		ip:               WildcardAddress,
		port:             int(server.Port.Number),
		rds:              server.Port.Name,
		useRemoteAddress: true,
		direction:        IngressTraceOperation,
		outboundListener: false,
		store:            config,
	}

	switch strings.ToUpper(server.Port.Protocol) {
	case "HTTP":
		return buildHTTPListener(opts)
	case "HTTPS":
		listener := buildHTTPListener(opts)
		listener.SSLContext = tlsToSSLContext(server.Tls, server.Port.Protocol)
		return listener
	case "GRPC", "HTTP2":
		listener := buildHTTPListener(opts)
		if server.Tls != nil {
			listener.SSLContext = tlsToSSLContext(server.Tls, server.Port.Protocol)
		}
		return listener
	case "TCP":
		log.Warnf("TCP protocol support for Gateways is not yet implemented")
		return nil
	case "MONGO":
		log.Warnf("Mongo protocol support for Gateways is not yet implemented")
		return nil
	default:
		log.Warnf("Gateway with invalid protocol: %q; %v", server.Port.Protocol, server)
		return nil
	}
}

// TODO: this isn't really correct: we need xDS v2 APIs to really configure this correctly.
// Our TLS options align with SDSv2 DownstreamTlsContext, but the v1 API's SSLContext is split
// into three pieces; we need at least two of the pieces here.
func tlsToSSLContext(tls *routing.Server_TLSOptions, protocol string) *SSLContext {
	return &SSLContext{
		CertChainFile:            tls.ServerCertificate,
		PrivateKeyFile:           tls.PrivateKey,
		CaCertFile:               tls.CaCertificates,
		RequireClientCertificate: tls.Mode == routing.Server_TLSOptions_MUTUAL,
		ALPNProtocols:            strings.Join(ListenersALPNProtocols, ","),
	}
}

// buildGatewayHTTPRoutes creates HTTP route configs indexed by ports for the
// traffic outbound from the proxy instance
func buildGatewayHTTPRoutes(mesh *meshconfig.MeshConfig, sidecar model.Node,
	instances []*model.ServiceInstance, services []*model.Service, config model.IstioConfigStore, targetGateways map[string]bool) HTTPRouteConfigs {
	httpConfigs := make(HTTPRouteConfigs)
	suffix := strings.Split(sidecar.Domain, ".")

	// outbound connections/requests are directed to service ports; we create a
	// map for each service port to define filters
	for _, service := range services {
		for _, servicePort := range service.Ports {
			routes := buildDestinationHTTPRoutesWithGateways(sidecar, service, servicePort, instances, config, buildOutboundCluster, targetGateways)

			if len(routes) > 0 {
				host := buildVirtualHost(service, servicePort, suffix, routes)
				http := httpConfigs.EnsurePort(servicePort.Port)

				// there should be at most one occurrence of the service for the same
				// port since service port values are distinct; that means the virtual
				// host domains, which include the sole domain name for the service, do
				// not overlap for the same route config.
				// for example, a service "a" with two ports 80 and 8080, would have virtual
				// hosts on 80 and 8080 listeners that contain domain "a".
				http.VirtualHosts = append(http.VirtualHosts, host)
			}
		}
	}

	return httpConfigs.normalize()
}
