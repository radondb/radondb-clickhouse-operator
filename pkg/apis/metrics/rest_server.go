// Copyright 2020 [RadonDB](https://github.com/radondb). All rights reserved.
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

package metrics

import (
	"net/http"

	log "github.com/golang/glog"
	// log "k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartMetricsREST start Prometheus metrics exporter in background
func StartMetricsREST(
	chAccess *CHAccessInfo,

	metricsAddress string,
	metricsPath string,
) {
	log.V(1).Infof("Starting metrics exporter at '%s%s'\n", metricsAddress, metricsPath)

	exporter = NewExporter(chAccess)
	prometheus.MustRegister(exporter)

	http.Handle(metricsPath, promhttp.Handler())

	go http.ListenAndServe(metricsAddress, nil)
}
