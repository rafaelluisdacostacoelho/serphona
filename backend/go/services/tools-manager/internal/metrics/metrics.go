package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ToolsReads = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tools_manager_tools_reads_total",
		Help: "Count of tool list/read operations",
	}, []string{"tenant"})
	ToolsWrites = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tools_manager_tools_writes_total",
		Help: "Count of tool write operations",
	}, []string{"tenant"})
	CatalogReads = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tools_manager_catalog_reads_total",
		Help: "Count of catalog resolved reads",
	}, []string{"tenant"})
	SecretFetches = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tools_manager_secret_fetches_total",
		Help: "Count of secret fetch operations",
	}, []string{"tenant", "cache_hit"})
	Errors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tools_manager_errors_total",
		Help: "Count of error responses",
	}, []string{"tenant", "path", "code"})
)

// Register all metrics with Prometheus registry.
func Register(reg prometheus.Registerer) {
	reg.MustRegister(ToolsReads, ToolsWrites, CatalogReads, SecretFetches, Errors)
}
