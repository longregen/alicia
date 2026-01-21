package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "alicia_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "alicia_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	ConversationsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "alicia_conversations_active",
		Help: "Number of active conversations",
	})

	MessagesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "alicia_messages_total",
		Help: "Total messages processed",
	})

	LLMRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "alicia_llm_requests_total",
		Help: "Total LLM requests",
	}, []string{"model", "status"})

	LLMRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "alicia_llm_request_duration_seconds",
		Help:    "LLM request duration",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
	}, []string{"model"})

	ASRRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "alicia_asr_request_duration_seconds",
		Help:    "ASR transcription duration",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
	})

	TTSRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "alicia_tts_request_duration_seconds",
		Help:    "TTS synthesis duration",
		Buckets: []float64{0.1, 0.5, 1, 2, 5},
	})
)
