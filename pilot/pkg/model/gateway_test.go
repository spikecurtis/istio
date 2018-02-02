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
	"testing"

	routing "istio.io/api/routing/v1alpha2"
)

var (
	tlsOne = &routing.Server_TLSOptions{
		HttpsRedirect:     true,
		Mode:              routing.Server_TLSOptions_SIMPLE,
		ServerCertificate: "server.pem",
		PrivateKey:        "key.pem",
	}
	tlsTwo = &routing.Server_TLSOptions{
		HttpsRedirect: false,
	}

	port80 = &routing.Port{
		Number:   80,
		Name:     "http-foo",
		Protocol: "HTTP",
	}
	port443 = &routing.Port{
		Number:   443,
		Name:     "https-foo",
		Protocol: "HTTPS",
	}
)

func TestMergeGateways(t *testing.T) {
	tests := []struct {
		name string
		b    *routing.Gateway
		a    *routing.Gateway
		out  *routing.Gateway
	}{
		{"idempotent",
			&routing.Gateway{Servers: []*routing.Server{}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port443,
					Tls:   tlsOne,
					Hosts: []string{"example.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port443,
					Tls:   tlsOne,
					Hosts: []string{"example.com"},
				},
			}},
		},
		{"different ports",
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"example.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port443,
					Tls:   tlsOne,
					Hosts: []string{"example.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"example.com"},
				}, {
					Port:  port443,
					Tls:   tlsOne,
					Hosts: []string{"example.com"},
				}}},
		},
		{"same ports different domains",
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"foo.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"bar.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"bar.com", "foo.com"},
				},
			}},
		},
		{"different domains, different ports",
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"foo.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port443,
					Tls:   tlsOne,
					Hosts: []string{"bar.com"},
				},
			}},
			&routing.Gateway{Servers: []*routing.Server{
				{
					Port:  port80,
					Tls:   tlsOne,
					Hosts: []string{"foo.com"},
				}, {
					Port:  port443,
					Tls:   tlsOne,
					Hosts: []string{"bar.com"},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// we want to save the original state of tt.a for printing if we fail the test, so we'll merge into a new gateway struct.
			actual := &routing.Gateway{}
			MergeGateways(actual, tt.a)
			MergeGateways(actual, tt.b)
			if !reflect.DeepEqual(actual, tt.out) {
				t.Fatalf("MergeGateways(%v, %v); got %v, wanted %v", tt.a, tt.b, actual, tt.out)
			}
		})
	}

}

func TestServersEqual(t *testing.T) {
	tests := []struct {
		name  string
		b     *routing.Server
		a     *routing.Server
		equal bool
	}{
		{"empty",
			&routing.Server{Port: &routing.Port{}, Tls: &routing.Server_TLSOptions{}},
			&routing.Server{Port: &routing.Port{}, Tls: &routing.Server_TLSOptions{}},
			true},
		{"happy",
			&routing.Server{Port: port80, Tls: tlsOne, Hosts: []string{"test.com"}},
			&routing.Server{Port: port80, Tls: tlsOne, Hosts: []string{"example.com"}},
			true},
		{"different ports",
			&routing.Server{Port: port80, Tls: tlsTwo},
			&routing.Server{Port: port443, Tls: tlsTwo},
			false},
		{"different tls",
			&routing.Server{Port: port80, Tls: tlsOne},
			&routing.Server{Port: port80, Tls: tlsTwo},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := serversEqual(tt.a, tt.b)
			if actual != tt.equal {
				t.Fatalf("ServersEqual(%v, %v) = %t, wanted %v", tt.a, tt.b, actual, tt.equal)
			}
		})
	}
}

func TestPortsEqual(t *testing.T) {
	tests := []struct {
		name  string
		b     *routing.Port
		a     *routing.Port
		equal bool
	}{
		{"empty", &routing.Port{}, &routing.Port{}, true},
		{"happy",
			&routing.Port{Number: 1, Name: "Bill", Protocol: "HTTP"},
			&routing.Port{Number: 1, Name: "Bob", Protocol: "HTTP"},
			true},
		{"case insensitive",
			&routing.Port{Number: 1, Protocol: "GRPC"},
			&routing.Port{Number: 1, Protocol: "grpc"},
			true},
		{"different numbers",
			&routing.Port{Number: 1, Protocol: "tcp"},
			&routing.Port{Number: 2, Protocol: "tcp"},
			false},
		{"different protocols",
			&routing.Port{Number: 1, Protocol: "http2"},
			&routing.Port{Number: 1, Protocol: "http"},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := portsEqual(tt.a, tt.b)
			if actual != tt.equal {
				t.Fatalf("portsEqual(%v, %v) = %t, wanted %v", tt.a, tt.b, actual, tt.equal)
			}
		})
	}
}

func TestTlsEqual(t *testing.T) {
	tests := []struct {
		name  string
		b     *routing.Server_TLSOptions
		a     *routing.Server_TLSOptions
		equal bool
	}{
		{"empty", &routing.Server_TLSOptions{}, &routing.Server_TLSOptions{}, true},
		{"happy",
			&routing.Server_TLSOptions{HttpsRedirect: true, Mode: routing.Server_TLSOptions_SIMPLE, ServerCertificate: "server.pem", PrivateKey: "key.pem"},
			&routing.Server_TLSOptions{HttpsRedirect: true, Mode: routing.Server_TLSOptions_SIMPLE, ServerCertificate: "server.pem", PrivateKey: "key.pem"},
			true},
		{"different",
			&routing.Server_TLSOptions{HttpsRedirect: true, Mode: routing.Server_TLSOptions_SIMPLE, ServerCertificate: "server.pem", PrivateKey: "key.pem"},
			&routing.Server_TLSOptions{HttpsRedirect: false},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tlsEqual(tt.a, tt.b)
			if actual != tt.equal {
				t.Fatalf("tlsEqual(%v, %v) = %t, wanted %v", tt.a, tt.b, actual, tt.equal)
			}
		})
	}
}
