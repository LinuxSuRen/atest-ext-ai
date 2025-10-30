/*
Copyright 2025 API Testing Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package metrics provides Prometheus metrics for monitoring AI operations
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AI请求计数
	aiRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atest_ai_requests_total",
			Help: "Total number of AI requests",
		},
		[]string{"method", "provider", "status"},
	)

	// AI请求延迟
	aiRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "atest_ai_request_duration_seconds",
			Help:    "AI request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
		},
		[]string{"method", "provider"},
	)

	// AI服务健康状态
	aiServiceHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atest_ai_service_health",
			Help: "AI service health status (1=healthy, 0=unhealthy)",
		},
		[]string{"provider"},
	)
)

// RecordRequest 记录AI请求
func RecordRequest(method, provider, status string) {
	aiRequestsTotal.WithLabelValues(method, provider, status).Inc()
}

// RecordDuration 记录请求延迟
func RecordDuration(method, provider string, duration float64) {
	aiRequestDuration.WithLabelValues(method, provider).Observe(duration)
}

// SetHealthStatus 设置健康状态
func SetHealthStatus(provider string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	aiServiceHealth.WithLabelValues(provider).Set(value)
}
