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

package app

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	log "github.com/golang/glog"
	// log "k8s.io/klog"

	"github.com/radondb/clickhouse-operator/pkg/apis/metrics"
	"github.com/radondb/clickhouse-operator/pkg/version"
)

// Prometheus exporter defaults
const (
	defaultMetricsEndpoint = ":8888"

	metricsPath = "/metrics"

	defaultPort     = 8123
	defaultPassword = ""
	defaultUsername = "default"
)

// CLI parameter variables
var (
	// versionRequest defines request for clickhouse-operator version report. Operator should exit after version printed
	versionRequest bool

	// chopConfigFile defines path to clickhouse-operator config file to be used
	chopConfigFile string

	// kubeConfigFile defines path to kube config file to be used
	kubeConfigFile string

	// masterURL defines URL of kubernetes master to be used
	masterURL string

	// metricsEP defines metrics end-point IP address
	metricsEP string

	// port to access clickhouse
	port int

	// username to access clickhouse
	username string

	// password to access clickhouse
	password string
)

func init() {
	flag.BoolVar(&versionRequest, "version", false, "Display clickhouse-operator version and exit")
	flag.StringVar(&chopConfigFile, "config", "", "Path to clickhouse-operator config file.")
	flag.StringVar(&kubeConfigFile, "kubeconfig", "", "Path to custom kubernetes config file. Makes sense if runs outside of the cluster only.")
	flag.StringVar(&masterURL, "master", "", "The address of custom Kubernetes API server. Makes sense if runs outside of the cluster and not being specified in kube config file only.")
	flag.StringVar(&metricsEP, "metrics-endpoint", defaultMetricsEndpoint, "The Prometheus exporter endpoint.")
	flag.IntVar(&port, "clickhouse-port", defaultPort, "The Prometheus exporter endpoint.")
	flag.StringVar(&username, "clickhouse-username", defaultUsername, "The Prometheus exporter endpoint.")
	flag.StringVar(&password, "clickhouse-password", defaultPassword, "The Prometheus exporter endpoint.")

	if u := os.Getenv("CLICKHOUSE_USERNAME"); u != "" {
		username = u
	}
	if p := os.Getenv("CLICKHOUSE_PASSWORD"); p != "" {
		password = p
	}
	if p := os.Getenv("CLICKHOUSE_PORT"); p != "" {
		port, _ = strconv.Atoi(p)
	}

	flag.Parse()
}

// Run is an entry point of the application
func Run() {
	if versionRequest {
		fmt.Printf("%s\n", version.Version)
		os.Exit(0)
	}

	// Set OS signals and termination context
	ctx, cancelFunc := context.WithCancel(context.Background())
	stopChan := make(chan os.Signal, 2)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopChan
		cancelFunc()
		<-stopChan
		os.Exit(1)
	}()

	log.Infof("Starting metrics exporter. Version:%s GitSHA:%s BuiltAt:%s\n", version.Version, version.GitSHA, version.BuiltAt)

	metrics.StartMetricsREST(
		metrics.NewCHAccessInfo(
			username,
			password,
			port,
		),

		metricsEP,
		metricsPath,
	)

	<-ctx.Done()
}
