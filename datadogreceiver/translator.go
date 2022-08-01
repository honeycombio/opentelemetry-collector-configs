// Copyright 2021, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datadogreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/datadogreceiver"

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	datadogpb "github.com/DataDog/datadog-agent/pkg/trace/exportable/pb"
	"github.com/tinylib/msgp/msgp"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/collector/semconv/v1.6.1"
)

func addResourceData(req *http.Request, rs *pcommon.Resource) {
	attrs := rs.Attributes()
	attrs.Clear()
	attrs.EnsureCapacity(3)
	attrs.UpsertString("telemetry.sdk.name", "Datadog")
	attrs.UpsertString("telemetry.sdk.version", version+"/"+"Datadog-"+req.Header.Get("Datadog-Meta-Tracer-Version"))
	attrs.UpsertString("telemetry.sdk.language", req.Header.Get("Datadog-Meta-Lang"))
}
func toTraces(traces datadogpb.Traces, req *http.Request) ptrace.Traces {

	dest := ptrace.NewTraces()
	resSpans := dest.ResourceSpans()
	rspan := resSpans.AppendEmpty()
	resource := rspan.Resource()
	addResourceData(req, &resource)

	ils := rspan.ScopeSpans().AppendEmpty()

	ils.Scope().SetName("Datadog-" + req.Header.Get("Datadog-Meta-Lang"))
	ils.Scope().SetVersion(req.Header.Get("Datadog-Meta-Tracer-Version"))

	for _, trace := range traces {

		spans := ptrace.NewSpanSlice()
		spans.EnsureCapacity(len(trace))
		for _, span := range trace {
			newSpan := spans.AppendEmpty()
			newSpan.SetTraceID(uInt64ToTraceID(0, span.TraceID))
			newSpan.SetSpanID(uInt64ToSpanID(span.SpanID))
			newSpan.SetStartTimestamp(pcommon.Timestamp(span.Start))
			newSpan.SetEndTimestamp(pcommon.Timestamp(span.Start + span.Duration))
			newSpan.SetParentSpanID(uInt64ToSpanID(span.ParentID))
			newSpan.SetName(span.Resource)

			if span.Error > 0 {
				newSpan.Status().SetCode(ptrace.StatusCodeError)
			} else {
				newSpan.Status().SetCode(ptrace.StatusCodeOk)
			}

			validattributes := make(map[string]string)
			for k, v := range span.GetMeta() {
				k = translateDataDogKeyToOtel(k)
				if len(k) > 0 {
					validattributes[k] = v
				}
			}

			attrs := newSpan.Attributes()
			attrs.EnsureCapacity(len(validattributes) + 2)
			attrs.InsertString(semconv.AttributeServiceName, span.Service)
			attrs.InsertString("resource", span.Name)
			for k, v := range validattributes {
				attrs.InsertString(k, v)
			}

			switch span.Type {
			case "web":
				newSpan.SetKind(ptrace.SpanKindServer)
			case "custom":
				newSpan.SetKind(ptrace.SpanKindUnspecified)
			default:
				newSpan.SetKind(ptrace.SpanKindClient)
			}
		}
		spans.MoveAndAppendTo(ils.Spans())
	}

	return dest
}

func translateDataDogKeyToOtel(k string) string {
	// Tags prefixed with _dd. are for Datadog's use only, adding them to another backend will just increase cardinality needlessly
	if strings.HasPrefix(k, "_dd.") {
		return ""
	}
	switch strings.ToLower(k) {
	case "env":
		return semconv.AttributeDeploymentEnvironment
	case "version":
		return semconv.AttributeServiceVersion
	case "container_id":
		return semconv.AttributeContainerID
	case "container_name":
		return semconv.AttributeContainerName
	case "image_name":
		return semconv.AttributeContainerImageName
	case "image_tag":
		return semconv.AttributeContainerImageTag
	case "process_id":
		return semconv.AttributeProcessPID
	case "error.stacktrace":
		return semconv.AttributeExceptionStacktrace
	case "error.msg":
		return semconv.AttributeExceptionMessage
	default:
		return k
	}

}

type InvalidMediaTypeError struct {
    Type string
}

func (e *InvalidMediaTypeError) Error() string {
    return fmt.Sprintf("Media type %s is not supported", e.Type)
}

func decodeRequest(req *http.Request, dest *datadogpb.Traces) error {
	switch mediaType := req.Header.Get("Content-Type"); mediaType {
	case "application/msgpack":
		if strings.HasPrefix(req.URL.Path, "/v0.5") {
			reader := datadogpb.NewMsgpReader(req.Body)
			defer datadogpb.FreeMsgpReader(reader)
			return dest.DecodeMsgDictionary(reader)
		}
		return msgp.Decode(req.Body, dest)
	case "application/json":
		return json.NewDecoder(req.Body).Decode(dest)
	default:
		return &InvalidMediaTypeError { mediaType }
	}
}

func uInt64ToTraceID(high, low uint64) pcommon.TraceID {
	traceID := [16]byte{}
	binary.BigEndian.PutUint64(traceID[:8], high)
	binary.BigEndian.PutUint64(traceID[8:], low)
	return pcommon.NewTraceID(traceID)
}

func uInt64ToSpanID(id uint64) pcommon.SpanID {
	spanID := [8]byte{}
	binary.BigEndian.PutUint64(spanID[:], id)
	return pcommon.NewSpanID(spanID)
}
