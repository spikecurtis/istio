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

package model

import (
	"reflect"
	"strings"

	routing "istio.io/api/routing/v1alpha2"
)

// MergeGateways takes two Gateways and combines their Servers. When two servers are the same (serversEqual == true)
// the hosts exposed by both servers are merged. Otherwise, the server is added to dst's server set.
func MergeGateways(dst, src *routing.Gateway) {
	// Simplify the loop logic below by handling the case where either Gateway is empty.
	if len(dst.Servers) == 0 {
		dst.Servers = src.Servers
		return
	} else if len(src.Servers) == 0 {
		return
	}

	servers := make([]*routing.Server, 0, len(dst.Servers))
	for _, ss := range src.Servers {
		for _, ds := range dst.Servers {
			if serversEqual(ss, ds) {
				ds.Hosts = append(ds.Hosts, ss.Hosts...)
			} else {
				servers = append(servers, ss)
			}
			servers = append(servers, ds)
		}
	}
	dst.Servers = servers
}

func serversEqual(a, b *routing.Server) bool {
	return portsEqual(a.Port, b.Port) && tlsEqual(a.Tls, b.Tls)
}

// Two ports are equal if they expose the same protocol on the same port number.
func portsEqual(a, b *routing.Port) bool {
	return a.Number == b.Number && strings.ToLower(a.Protocol) == strings.ToLower(b.Protocol)
}

// Two TLS Options are equal if all of their fields are equal.
func tlsEqual(a, b *routing.Server_TLSOptions) bool {
	return reflect.DeepEqual(a, b)
}
