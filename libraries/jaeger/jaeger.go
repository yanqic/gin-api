package jaeger

import (
	"gin-api/app_const"
	"gin-api/libraries/logging"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	opentracing_log "github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

const (
	FIELD_LOG_ID       = "Log-Id"
	FIELD_TRACE_ID     = "Trace-Id"
	FIELD_SPAN_ID      = "Span-Id"
	FIELD_TRACER       = "Tracer"
	FIELD_SPAN_CONTEXT = "Span"
)

const (
	OPERATION_TYPE_HTTP     = "HTTP"
	OPERATION_TYPE_RPC      = "RPC"
	OPERATION_TYPE_MYSQL    = "MySQL"
	OPERATION_TYPE_REDIS    = "Redis"
	OPERATION_TYPE_RabbitMQ = "RabbitMQ"
)

func NewJaegerTracer(jaegerHostPort string) (opentracing.Tracer, io.Closer, error) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const", //固定采样
			Param: 1,       //1=全采样、0=不采样
		},

		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: jaegerHostPort,
		},

		ServiceName: app_const.SERVICE_NAME,
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return tracer, closer, nil
}

func Inject(c *gin.Context, header http.Header, operationName, operationType string) (opentracingSpan opentracing.Span, err error) {
	tracerInterface, ok := c.Get(FIELD_TRACER)
	if !ok {
		return
	}
	tracer, ok := tracerInterface.(opentracing.Tracer)
	if !ok {
		return
	}
	parentSpanInterface, ok := c.Get(FIELD_SPAN_CONTEXT)
	if !ok {
		return
	}
	parentSpanContext, ok := parentSpanInterface.(opentracing.SpanContext)
	if !ok {
		return
	}

	opentracingSpan = opentracing.StartSpan(
		operationName,
		opentracing.ChildOf(parentSpanContext),
		opentracing.Tag{Key: string(ext.Component), Value: operationType},
		ext.SpanKindRPCClient,
	)
	SetTag(c, opentracingSpan, parentSpanContext)
	err = tracer.Inject(opentracingSpan.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(header))
	if err != nil {
		opentracingSpan.LogFields(opentracing_log.String("inject-error", err.Error()))
	}

	return
}

func SetTag(c *gin.Context, span opentracing.Span, spanContext opentracing.SpanContext) {
	jaegerSpanContext := spanContextToJaegerContext(spanContext)
	span.SetTag(FIELD_TRACE_ID, jaegerSpanContext.TraceID().String())
	span.SetTag(FIELD_SPAN_ID, jaegerSpanContext.SpanID().String())
	span.SetTag(FIELD_LOG_ID, logging.ValueLogID(c))
}

func spanContextToJaegerContext(spanContext opentracing.SpanContext) jaeger.SpanContext {
	if sc, ok := spanContext.(jaeger.SpanContext); ok {
		return sc
	} else {
		return jaeger.SpanContext{}
	}
}
