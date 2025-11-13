package filterremapprocessor

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/tf-rob-w/otel-collector-filter-remap-processor/filterremapprocessor/internal/utils"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	// "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
)

type OverflowStrategy string

const (
	OverflowDrop    OverflowStrategy = "drop"
	OverflowForward OverflowStrategy = "forward"
)

// Config is the configuration for the filter remap processor. (filterprocessor says it's the configuration for Resource processor? Need to figure out what this means...)
type Config struct {
	ErrorMode ottl.ErrorMode `mapstructure:"error_mode"`

	// The amount of time a trace is retained in the processor before it's spans are remapped and sent to the next consumer, regardless of when the last span was seen (in seconds)
	MaxTraceRetention time.Duration `mapstructure:"max_trace_retention"`

	// Time to wait after the last span is seen before remapping and forwarding the trace (in seconds)
	LastSpanTimeout time.Duration `mapstructure:"last_span_timeout"`

	Traces TraceFilters `mapstructure:"traces"`

	// NumTraces is the number of traces kept on memory. Typically most of the data
	// of a trace is released after a sampling decision is taken.
	NumTraces uint64 `mapstructure:"num_traces"`
	// ExpectedNewTracesPerSec sets the expected number of new traces sending to the tail sampling processor
	// per second. This helps with allocating data structures with closer to actual usage size.
	ExpectedNewTracesPerSec uint64 `mapstructure:"expected_new_traces_per_sec"`
	// ExpectedAverageSpansPerTrace is used to allocate data structures for processing traces
	ExpectedAverageSpansPerTrace uint64 `mapstructure:"expected_average_spans_per_trace"`
	// If a root span matches the filter, it will be dropped. Defaults to false
	DropRootSpans bool `mapstructure:"drop_root_spans"`
	// If a span comes in after the last span timeout, we've already forwarded the trace to the next consumer and do not have the spans parent to know whether the parent was dropped or retained.
	// By default, we will just forward any orphaned spans without remapping them, but if this is set to true, we will remap the orphaned spans to root spans.
	RemapOrphanedSpans bool `mapstructure:"remap_orphaned_spans"`
	FlushOnShutdown    bool `mapstructure:"flush_on_shutdown"`
	// Forward queue size is the number of traces that can be queued for remapping and forwarding to the next consumer
	// Optional, default is 2 * ExpectedNewTracesPerSec
	// Recommended to test with default value first with expected workload, then adjust based on p99 of forward trace latency
	ForwardQueueSize uint64 `mapstructure:"forward_queue_size"`
	// Forward worker concurrency is the number of worker threads that will be used to remap and forward traces to the next consumer
	// Optional, default is min(8, GOMAXPROCS)
	ForwardWorkerConcurrency int `mapstructure:"forward_worker_concurrency"`
	// How to handle forward queue overflows (more traces added to queue than ForwardQueueSize)
	// Options - drop|forward
	// drop (default, recommended) - if the forward trace queue is full, drop the trace entirely, will result in lost data.
	// forward - if the forward trace queue is full, the processor will forward the trace to the next consumer by creating a new goroutine to forward the trace, if volume is too high this could result in a lot of goroutines being created.
	// Recommended approach to prevent data loss is to carefully configure ForwardQueueSize and ForwardWorkerConcurrency to match the expected workload.
	// Note: if the volume is too high, the processor will still drop traces if the overflow strategy is set to drop.
	OverflowStrategy OverflowStrategy `mapstructure:"overflow_strategy"`
}

type TraceFilters struct {
	SpanConditions []string `mapstructure:"span"`

	SpanEventConditions []string `mapstructure:"spanevent"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	var errs error

	if cfg.Traces.SpanConditions != nil {
		_, err := utils.NewBoolExprForSpan(cfg.Traces.SpanConditions, utils.StandardSpanFuncs(), ottl.PropagateError, component.TelemetrySettings{Logger: zap.NewNop()})
		errs = multierr.Append(errs, err)
	}

	if cfg.Traces.SpanEventConditions != nil {
		_, err := utils.NewBoolExprForSpanEvent(cfg.Traces.SpanEventConditions, utils.StandardSpanEventFuncs(), ottl.PropagateError, component.TelemetrySettings{Logger: zap.NewNop()})
		errs = multierr.Append(errs, err)
	}

	if cfg.MaxTraceRetention <= 0 {
		err := errors.New("trace_retention must be greater than 0")
		errs = multierr.Append(errs, err)
	}

	if cfg.MaxTraceRetention > 10*time.Minute {
		err := errors.New("trace_retention should not exceed 10 minutes")
		errs = multierr.Append(errs, err)
	}

	if cfg.LastSpanTimeout <= 0 {
		err := errors.New("last_span_timeout must be greater than 0")
		errs = multierr.Append(errs, err)
	}

	if cfg.ExpectedAverageSpansPerTrace < 1 {
		err := errors.New("expected_average_spans_per_trace must be greater than 0")
		errs = multierr.Append(errs, err)
	}

	if cfg.NumTraces < 1 {
		err := errors.New("num_traces must be greater than 0")
		errs = multierr.Append(errs, err)
	}

	if cfg.ForwardQueueSize < 0 {
		err := errors.New("forward_queue_size must be greater than 0")
		errs = multierr.Append(errs, err)
	}

	if cfg.ForwardWorkerConcurrency < 0 {
		err := errors.New("forward_worker_concurrency must be greater than 0")
		errs = multierr.Append(errs, err)
	}

	if cfg.OverflowStrategy != OverflowDrop && cfg.OverflowStrategy != OverflowForward {
		err := errors.New("overflow_strategy must be either 'drop' or 'forward'")
		errs = multierr.Append(errs, err)
	}

	return errs
}
