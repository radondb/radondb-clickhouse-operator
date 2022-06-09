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
	"time"

	log "github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultTimeout = 30 * time.Second
)

// Exporter implements prometheus.Collector interface
type Exporter struct {
	chAccessInfo *CHAccessInfo
	timeout      time.Duration
}

var exporter *Exporter

// NewExporter returns a new instance of Exporter type
func NewExporter(chAccess *CHAccessInfo) *Exporter {
	return &Exporter{
		chAccessInfo: chAccess,
		timeout:      defaultTimeout,
	}
}

// newFetcher returns new Metrics Fetcher for specified host
func (e *Exporter) newFetcher(hostname string) *ClickHouseFetcher {
	return NewClickHouseFetcher(
		hostname,
		e.chAccessInfo.Username,
		e.chAccessInfo.Password,
		e.chAccessInfo.Port,
	).SetQueryTimeout(e.timeout)
}

// Describe implements prometheus.Collector Describe method
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(e, ch)
}

// Collect implements prometheus.Collector Collect method
func (e *Exporter) Collect(c chan<- prometheus.Metric) {
	if c == nil {
		log.V(2).Info("Prometheus channel is closed. Skipping")
		return
	}

	log.V(2).Info("Starting Collect")

	fetcher := e.newFetcher("localhost")
	writer := NewPrometheusWriter(c)

	log.V(2).Infof("Querying metrics\n")
	if metrics, err := fetcher.getClickHouseQueryMetrics(); err == nil {
		log.V(2).Infof("Extracted %d metrics\n", len(metrics))
		writer.WriteMetrics(metrics)
		writer.WriteOKFetch("system.metrics")
	} else {
		// In case of an error fetching data from clickhouse store CHI name in e.cleanup
		log.V(2).Infof("Error querying metrics: %s\n", err)
		writer.WriteErrorFetch("system.metrics")
	}

	log.V(2).Infof("Querying table sizes\n")
	if tableSizes, err := fetcher.getClickHouseQueryTableSizes(); err == nil {
		log.V(2).Infof("Extracted %d table sizes\n", len(tableSizes))
		writer.WriteTableSizes(tableSizes)
		writer.WriteOKFetch("table sizes")
	} else {
		// In case of an error fetching data from clickhouse store CHI name in e.cleanup
		log.V(2).Infof("Error querying table sizes: %s\n", err)
		writer.WriteErrorFetch("table sizes")
	}

	log.V(2).Infof("Querying system replicas\n")
	if systemReplicas, err := fetcher.getClickHouseQuerySystemReplicas(); err == nil {
		log.V(2).Infof("Extracted %d system replicas\n", len(systemReplicas))
		writer.WriteSystemReplicas(systemReplicas)
		writer.WriteOKFetch("system.replicas")
	} else {
		// In case of an error fetching data from clickhouse store CHI name in e.cleanup
		log.V(2).Infof("Error querying system replicas: %s\n", err)
		writer.WriteErrorFetch("system.replicas")
	}

	log.V(2).Infof("Querying mutations\n")
	if mutations, err := fetcher.getClickHouseQueryMutations(); err == nil {
		log.V(2).Infof("Extracted %d mutations\n", len(mutations))
		writer.WriteMutations(mutations)
		writer.WriteOKFetch("system.mutations")
	} else {
		// In case of an error fetching data from clickhouse store CHI name in e.cleanup
		log.V(2).Infof("Error querying mutations: %s\n", err)
		writer.WriteErrorFetch("system.mutations")
	}

	log.V(2).Infof("Querying disks\n")
	if disks, err := fetcher.getClickHouseQuerySystemDisks(); err == nil {
		log.V(2).Infof("Extracted %d disks\n", len(disks))
		writer.WriteSystemDisks(disks)
		writer.WriteOKFetch("system.disks")
	} else {
		// In case of an error fetching data from clickhouse store CHI name in e.cleanup
		log.V(2).Infof("Error querying disks: %s\n", err)
		writer.WriteErrorFetch("system.disks")
	}

	log.V(2).Infof("Querying detached parts\n")
	if detachedParts, err := fetcher.getClickHouseQueryDetachedParts(); err == nil {
		log.V(2).Infof("Extracted %d detached parts info\n", len(detachedParts))
		writer.WriteDetachedParts(detachedParts)
		writer.WriteOKFetch("system.detached_parts")
	} else {
		// In case of an error fetching data from clickhouse store CHI name in e.cleanup
		log.V(2).Infof("Error querying detached parts: %s\n", err)
		writer.WriteErrorFetch("system.detached_parts")
	}

	log.V(2).Info("Finished Collect")
}
