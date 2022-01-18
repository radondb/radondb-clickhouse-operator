package backup

const (
	// ClickHouse open ports names and values
	chDefaultTCPPortNumber = int32(9000)
)

const (
	// Default ClickHouse client docker image to be used
	defaultClickHouseClientDockerImage = "radondb/clickhouse-client:21.1.3.32"
)

const (
	imagePullPolicyAlways       = "Always"
	imagePullPolicyPullNever    = "Never"
	imagePullPolicyIfNotPresent = "IfNotPresent"
)
