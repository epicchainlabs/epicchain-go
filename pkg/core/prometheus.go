package core

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics for monitoring service.
var (
	// blockHeight prometheus metric.
	blockHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current index of processed block",
			Name:      "current_block_height",
			Namespace: "neogo",
		},
	)
	// persistedHeight prometheus metric.
	persistedHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current persisted block count",
			Name:      "current_persisted_height",
			Namespace: "neogo",
		},
	)
	// headerHeight prometheus metric.
	headerHeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Current header height",
			Name:      "current_header_height",
			Namespace: "neogo",
		},
	)
	// mempoolUnsortedTx prometheus metric.
	mempoolUnsortedTx = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help:      "Mempool unsorted transactions",
			Name:      "mempool_unsorted_tx",
			Namespace: "neogo",
		},
	)
)

func init() {
	prometheus.MustRegister(
		blockHeight,
		persistedHeight,
		headerHeight,
		mempoolUnsortedTx,
	)
}

func updatePersistedHeightMetric(pHeight uint32) {
	persistedHeight.Set(float64(pHeight))
}

func updateHeaderHeightMetric(hHeight uint32) {
	headerHeight.Set(float64(hHeight))
}

func updateBlockHeightMetric(bHeight uint32) {
	blockHeight.Set(float64(bHeight))
}

// updateMempoolMetrics updates metric of the number of unsorted txs inside the mempool.
func updateMempoolMetrics(unsortedTxnLen int) {
	mempoolUnsortedTx.Set(float64(unsortedTxnLen))
}
