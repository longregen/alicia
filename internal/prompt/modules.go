package prompt

import (
	"context"
	"fmt"

	"github.com/XiaoConstantine/dspy-go/pkg/core"
	"github.com/XiaoConstantine/dspy-go/pkg/modules"
)

// AliciaPredict wraps dspy-go Predict with Alicia integration
type AliciaPredict struct {
	*modules.Predict
	tracer  Tracer
	metrics MetricsCollector
}

// Option configures an AliciaPredict module
type Option func(*AliciaPredict)

// WithTracer sets a tracer for the module
func WithTracer(tracer Tracer) Option {
	return func(p *AliciaPredict) {
		p.tracer = tracer
	}
}

// WithMetrics sets a metrics collector for the module
func WithMetrics(metrics MetricsCollector) Option {
	return func(p *AliciaPredict) {
		p.metrics = metrics
	}
}

// NewAliciaPredict creates a new AliciaPredict module
func NewAliciaPredict(sig Signature, opts ...Option) *AliciaPredict {
	ap := &AliciaPredict{
		Predict: modules.NewPredict(sig.Signature),
	}

	for _, opt := range opts {
		opt(ap)
	}

	return ap
}

// Process executes the prediction with tracing and metrics
func (p *AliciaPredict) Process(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	// Pre-execution hooks
	var span Span
	if p.tracer != nil {
		span = p.tracer.StartSpan(ctx, "predict")
		defer span.End()
	}

	// Execute
	outputs, err := p.Predict.Process(ctx, inputs)

	// Post-execution metrics
	if p.metrics != nil {
		p.metrics.RecordExecution(span, inputs, outputs, err)
	}

	if err != nil {
		return nil, fmt.Errorf("predict process failed: %w", err)
	}

	return outputs, nil
}

// Tracer defines the interface for tracing module execution
type Tracer interface {
	StartSpan(ctx context.Context, name string) Span
}

// Span represents a traced execution span
type Span interface {
	End()
	SetError(err error)
	SetAttribute(key string, value any)
}

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordExecution(span Span, inputs, outputs map[string]any, err error)
}

// NoOpTracer is a tracer that does nothing
type NoOpTracer struct{}

func (t *NoOpTracer) StartSpan(ctx context.Context, name string) Span {
	return &NoOpSpan{}
}

// NoOpSpan is a span that does nothing
type NoOpSpan struct{}

func (s *NoOpSpan) End()                               {}
func (s *NoOpSpan) SetError(err error)                 {}
func (s *NoOpSpan) SetAttribute(key string, value any) {}

// NoOpMetrics is a metrics collector that does nothing
type NoOpMetrics struct{}

func (m *NoOpMetrics) RecordExecution(span Span, inputs, outputs map[string]any, err error) {}

// ToProgram wraps the AliciaPredict module in a core.Program for use with dspy-go optimizers
func (p *AliciaPredict) ToProgram(moduleName string) core.Program {
	modules := map[string]core.Module{
		moduleName: p.Predict,
	}

	forward := func(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
		// Convert inputs
		anyInputs := make(map[string]any, len(inputs))
		for k, v := range inputs {
			anyInputs[k] = v
		}

		// Process through the module
		outputs, err := p.Process(ctx, anyInputs)
		if err != nil {
			return nil, err
		}

		// Convert outputs
		result := make(map[string]interface{}, len(outputs))
		for k, v := range outputs {
			result[k] = v
		}
		return result, nil
	}

	return core.NewProgram(modules, forward)
}
