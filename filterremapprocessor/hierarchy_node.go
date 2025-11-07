package filterremapprocessor

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// spanAndScope is a structure for holding information about span and its instrumentation scope.
// required for preserving the instrumentation library information while sampling.
// We use pointers here to fast find the span in the map.
type spanAndScope struct {
	span                 *ptrace.Span
	instrumentationScope *pcommon.InstrumentationScope
	scopeSchemaUrl       string
	resource             *pcommon.Resource
	resourceSchemaUrl    string
}

type hierarchyNode interface {
	SpanID() pcommon.SpanID
	ParentID() pcommon.SpanID
	setParentId(pcommon.SpanID, bool)
	IsParentRetained() bool
}

type retainedNode interface {
	hierarchyNode
	SpanData() *spanAndScope
}

type retainedHierarchyNode struct {
	spanData       *spanAndScope
	parentRetained bool
}

type droppedHierarchyNode struct {
	spanId         pcommon.SpanID
	traceId        pcommon.TraceID
	parentSpanId   pcommon.SpanID
	parentRetained bool
}

func (n *retainedHierarchyNode) SpanID() pcommon.SpanID {
	return n.spanData.span.SpanID()
}

func (n *retainedHierarchyNode) ParentID() pcommon.SpanID {
	return n.spanData.span.ParentSpanID()
}

func (n *retainedHierarchyNode) SpanData() *spanAndScope {
	return n.spanData
}

func (n *retainedHierarchyNode) IsParentRetained() bool {
	return n.parentRetained
}

func (n *retainedHierarchyNode) setParentId(parentSpanId pcommon.SpanID, parentRetained bool) {
	n.spanData.span.SetParentSpanID(parentSpanId)
	n.parentRetained = parentRetained
}

func (n *droppedHierarchyNode) SpanID() pcommon.SpanID {
	return n.spanId
}

func (n *droppedHierarchyNode) ParentID() pcommon.SpanID {
	return n.parentSpanId
}

func (n *droppedHierarchyNode) IsParentRetained() bool {
	return n.parentRetained
}

func (n *droppedHierarchyNode) setParentId(parentSpanId pcommon.SpanID, parentRetained bool) {
	n.parentSpanId = parentSpanId
	n.parentRetained = parentRetained
}
