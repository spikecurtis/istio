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

package util

import (
	"testing"

	"istio.io/istio/pilot/pkg/model"
)

func TestGetNetworkEndpointAddress(t *testing.T) {
	neUnix := &model.NetworkEndpoint{
		Family:  model.AddressFamilyUnix,
		Address: "/var/run/test/test.sock",
	}
	aUnix := GetNetworkEndpointAddress(neUnix)
	if aUnix.GetPipe() == nil {
		t.Fatalf("GetAddress() => want Pipe, got %s", aUnix.String())
	}
	if aUnix.GetPipe().GetPath() != neUnix.Address {
		t.Fatalf("GetAddress() => want path %s, got %s", neUnix.Address, aUnix.GetPipe().GetPath())
	}

	neIP := &model.NetworkEndpoint{
		Family:  model.AddressFamilyTCP,
		Address: "192.168.10.45",
		Port:    4558,
	}
	aIP := GetNetworkEndpointAddress(neIP)
	sock := aIP.GetSocketAddress()
	if sock == nil {
		t.Fatalf("GetAddress() => want SocketAddress, got %s", aIP.String())
	}
	if sock.GetAddress() != neIP.Address {
		t.Fatalf("GetAddress() => want %s, got %s", neIP.Address, sock.GetAddress())
	}
	if int(sock.GetPortValue()) != neIP.Port {
		t.Fatalf("GetAddress() => want port %d, got port %d", neIP.Port, sock.GetPortValue())
	}
}

func TestValidateNetworkEndpointAddress(t *testing.T) {
	testCases := []struct {
		name  string
		ne    *model.NetworkEndpoint
		valid bool
	}{
		{
			"Unix OK",
			&model.NetworkEndpoint{Family: model.AddressFamilyUnix, Address: "/absolute/path"},
			true,
		},
		{
			"IP OK",
			&model.NetworkEndpoint{Address: "12.3.4.5", Port: 76},
			true,
		},
		{
			"Unix not absolute",
			&model.NetworkEndpoint{Family: model.AddressFamilyUnix, Address: "./socket"},
			false,
		},
		{
			"IP invalid",
			&model.NetworkEndpoint{Address: "260.3.4.5", Port: 76},
			false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNetworkEndpointAddress(tc.ne)
			if tc.valid && err != nil {
				t.Fatalf("ValidateAddress() => want error nil got %v", err)
			} else if !tc.valid && err == nil {
				t.Fatalf("ValidateAddress() => want error got nil")
			}
		})
	}
}
