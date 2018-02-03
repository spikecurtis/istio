// Copyright 2017 Istio Authors.
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
	"io/ioutil"
	"reflect"
	"testing"

	"encoding/json"

	"github.com/onsi/gomega"

	routing "istio.io/api/routing/v1alpha2"
	"istio.io/istio/pilot/pkg/config/memory"
	"istio.io/istio/pilot/pkg/model"
)

var (
	httpGateway = fileConfig{
		meta: model.ConfigMeta{Type: model.Gateway.Type, Name: "http"},
		file: "testdata/gateway-http.yaml",
	}

	httpGateway2 = fileConfig{
		meta: model.ConfigMeta{Type: model.Gateway.Type, Name: "http-also"},
		file: "testdata/gateway-http.yaml",
	}

	httpsGateway = fileConfig{
		meta: model.ConfigMeta{Type: model.Gateway.Type, Name: "https"},
		file: "testdata/gateway-https.yaml",
	}

	h2Gateway = fileConfig{
		meta: model.ConfigMeta{Type: model.Gateway.Type, Name: "h2"},
		file: "testdata/gateway-h2.yaml",
	}
)

func TestBuildGatewayListeners(t *testing.T) {
	tests := []struct {
		name   string
		in     []fileConfig
		golden string
	}{
		{"http", []fileConfig{httpGateway}, "testdata/gateway-http-listener.json.golden"},
		{"http, duplicates", []fileConfig{httpGateway, httpGateway2}, "testdata/gateway-http-listener.json.golden"},
		{"https", []fileConfig{httpsGateway}, "testdata/gateway-https-listener.json.golden"},
		//{"h2", []fileConfig{h2Gateway}, "testdata/gateway-h2-listener.json.golden"},
		{"http and h2", []fileConfig{httpGateway, h2Gateway}, "testdata/gateway-h2-listener.json.golden"},
		//{"http and https", []fileConfig{httpGateway, httpsGateway}, "testdata/gateway-h2-listener.json.golden"},
		{"https and h2", []fileConfig{httpsGateway, h2Gateway}, "testdata/gateway-https-h2-listener.json.golden"},
		//{"http, https, h2", []fileConfig{httpGateway, httpsGateway, h2Gateway}, "testdata/gateway-h2-listener.json.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

			registry := memory.Make(model.IstioConfigTypes)

			for _, file := range tt.in {
				addConfig(registry, file, t)
			}

			config := model.MakeIstioStore(registry)
			mesh := makeMeshConfig()

			mockIngress := model.Node{
				IPAddress: "10.3.3.3",
				ID:        "ingress.default",
			}

			listeners := buildGatewayListeners(&mesh, []*model.ServiceInstance{}, []*model.Service{}, config, mockIngress)
			out, err := json.Marshal(ldsResponse{Listeners: listeners})
			g.Expect(err).NotTo(gomega.HaveOccurred())

			contents, err := ioutil.ReadFile(tt.golden)
			g.Expect(err).NotTo(gomega.HaveOccurred())

			g.Expect(out).To(gomega.MatchJSON(contents))
		})
	}
}

func TestTlsToSSLContext(t *testing.T) {
	// TODO: flesh out tests once we move to SDSv2 types
	tests := []struct {
		name     string
		protocol string
		in       *routing.Server_TLSOptions
		out      *SSLContext
	}{
		{"empty", "", &routing.Server_TLSOptions{}, &SSLContext{ALPNProtocols: "h2,http/1.1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := tlsToSSLContext(tt.in, tt.protocol)
			if !reflect.DeepEqual(out, tt.out) {
				t.Fatalf("tlsToSSLContext(%v, %q) = %v, wanted %v", tt.in, tt.protocol, out, tt.out)
			}
		})
	}
}
