// Package oc supports OpenCensus tracing and metrics for the Go Cloud Development Kit.
package oc

import (
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// LatencyMeasure returns the measure for method call latency used
// by Go CDK APIs.
func LatencyMeasure(pkg string) *stats.Float64Measure {
	return stats.Float64(
		pkg+"/latency",
		"Latency of method call",
		stats.UnitMilliseconds)
}

// Tag keys used for the standard Go CDK views.
var (
	MethodKey   = tag.MustNewKey("gocdk_method")
	StatusKey   = tag.MustNewKey("gocdk_status")
	ProviderKey = tag.MustNewKey("gocdk_provider")
)

// Views returns the views supported by Go CDK APIs.
func Views(pkg string, latencyMeasure *stats.Float64Measure) []*view.View {
	return []*view.View{
		{
			Name:        pkg + "/completed_calls",
			Measure:     latencyMeasure,
			Description: "Count of method calls by provider, method and status.",
			TagKeys:     []tag.Key{ProviderKey, MethodKey, StatusKey},
			Aggregation: view.Count(),
		},
		{
			Name:        pkg + "/latency",
			Measure:     latencyMeasure,
			Description: "Distribution of method latency, by provider and method.",
			TagKeys:     []tag.Key{ProviderKey, MethodKey},
			Aggregation: ocgrpc.DefaultMillisecondsDistribution,
		},
	}
}
