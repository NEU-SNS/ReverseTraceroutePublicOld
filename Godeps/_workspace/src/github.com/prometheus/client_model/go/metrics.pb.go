/*
Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
// Code generated by protoc-gen-go.
// source: metrics.proto
// DO NOT EDIT!

/*
Package io_prometheus_client is a generated protocol buffer package.

It is generated from these files:
	metrics.proto

It has these top-level messages:
	LabelPair
	Gauge
	Counter
	Quantile
	Summary
	Untyped
	Histogram
	Bucket
	Metric
	MetricFamily
*/
package io_prometheus_client

import proto "github.com/golang/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type MetricType int32

const (
	MetricType_COUNTER   MetricType = 0
	MetricType_GAUGE     MetricType = 1
	MetricType_SUMMARY   MetricType = 2
	MetricType_UNTYPED   MetricType = 3
	MetricType_HISTOGRAM MetricType = 4
)

var MetricType_name = map[int32]string{
	0: "COUNTER",
	1: "GAUGE",
	2: "SUMMARY",
	3: "UNTYPED",
	4: "HISTOGRAM",
}
var MetricType_value = map[string]int32{
	"COUNTER":   0,
	"GAUGE":     1,
	"SUMMARY":   2,
	"UNTYPED":   3,
	"HISTOGRAM": 4,
}

func (x MetricType) Enum() *MetricType {
	p := new(MetricType)
	*p = x
	return p
}
func (x MetricType) String() string {
	return proto.EnumName(MetricType_name, int32(x))
}
func (x *MetricType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(MetricType_value, data, "MetricType")
	if err != nil {
		return err
	}
	*x = MetricType(value)
	return nil
}

type LabelPair struct {
	Name             *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Value            *string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *LabelPair) Reset()         { *m = LabelPair{} }
func (m *LabelPair) String() string { return proto.CompactTextString(m) }
func (*LabelPair) ProtoMessage()    {}

func (m *LabelPair) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *LabelPair) GetValue() string {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return ""
}

type Gauge struct {
	Value            *float64 `protobuf:"fixed64,1,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Gauge) Reset()         { *m = Gauge{} }
func (m *Gauge) String() string { return proto.CompactTextString(m) }
func (*Gauge) ProtoMessage()    {}

func (m *Gauge) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Counter struct {
	Value            *float64 `protobuf:"fixed64,1,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Counter) Reset()         { *m = Counter{} }
func (m *Counter) String() string { return proto.CompactTextString(m) }
func (*Counter) ProtoMessage()    {}

func (m *Counter) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Quantile struct {
	Quantile         *float64 `protobuf:"fixed64,1,opt,name=quantile" json:"quantile,omitempty"`
	Value            *float64 `protobuf:"fixed64,2,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Quantile) Reset()         { *m = Quantile{} }
func (m *Quantile) String() string { return proto.CompactTextString(m) }
func (*Quantile) ProtoMessage()    {}

func (m *Quantile) GetQuantile() float64 {
	if m != nil && m.Quantile != nil {
		return *m.Quantile
	}
	return 0
}

func (m *Quantile) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Summary struct {
	SampleCount      *uint64     `protobuf:"varint,1,opt,name=sample_count" json:"sample_count,omitempty"`
	SampleSum        *float64    `protobuf:"fixed64,2,opt,name=sample_sum" json:"sample_sum,omitempty"`
	Quantile         []*Quantile `protobuf:"bytes,3,rep,name=quantile" json:"quantile,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}

func (m *Summary) Reset()         { *m = Summary{} }
func (m *Summary) String() string { return proto.CompactTextString(m) }
func (*Summary) ProtoMessage()    {}

func (m *Summary) GetSampleCount() uint64 {
	if m != nil && m.SampleCount != nil {
		return *m.SampleCount
	}
	return 0
}

func (m *Summary) GetSampleSum() float64 {
	if m != nil && m.SampleSum != nil {
		return *m.SampleSum
	}
	return 0
}

func (m *Summary) GetQuantile() []*Quantile {
	if m != nil {
		return m.Quantile
	}
	return nil
}

type Untyped struct {
	Value            *float64 `protobuf:"fixed64,1,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Untyped) Reset()         { *m = Untyped{} }
func (m *Untyped) String() string { return proto.CompactTextString(m) }
func (*Untyped) ProtoMessage()    {}

func (m *Untyped) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Histogram struct {
	SampleCount      *uint64   `protobuf:"varint,1,opt,name=sample_count" json:"sample_count,omitempty"`
	SampleSum        *float64  `protobuf:"fixed64,2,opt,name=sample_sum" json:"sample_sum,omitempty"`
	Bucket           []*Bucket `protobuf:"bytes,3,rep,name=bucket" json:"bucket,omitempty"`
	XXX_unrecognized []byte    `json:"-"`
}

func (m *Histogram) Reset()         { *m = Histogram{} }
func (m *Histogram) String() string { return proto.CompactTextString(m) }
func (*Histogram) ProtoMessage()    {}

func (m *Histogram) GetSampleCount() uint64 {
	if m != nil && m.SampleCount != nil {
		return *m.SampleCount
	}
	return 0
}

func (m *Histogram) GetSampleSum() float64 {
	if m != nil && m.SampleSum != nil {
		return *m.SampleSum
	}
	return 0
}

func (m *Histogram) GetBucket() []*Bucket {
	if m != nil {
		return m.Bucket
	}
	return nil
}

type Bucket struct {
	CumulativeCount  *uint64  `protobuf:"varint,1,opt,name=cumulative_count" json:"cumulative_count,omitempty"`
	UpperBound       *float64 `protobuf:"fixed64,2,opt,name=upper_bound" json:"upper_bound,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Bucket) Reset()         { *m = Bucket{} }
func (m *Bucket) String() string { return proto.CompactTextString(m) }
func (*Bucket) ProtoMessage()    {}

func (m *Bucket) GetCumulativeCount() uint64 {
	if m != nil && m.CumulativeCount != nil {
		return *m.CumulativeCount
	}
	return 0
}

func (m *Bucket) GetUpperBound() float64 {
	if m != nil && m.UpperBound != nil {
		return *m.UpperBound
	}
	return 0
}

type Metric struct {
	Label            []*LabelPair `protobuf:"bytes,1,rep,name=label" json:"label,omitempty"`
	Gauge            *Gauge       `protobuf:"bytes,2,opt,name=gauge" json:"gauge,omitempty"`
	Counter          *Counter     `protobuf:"bytes,3,opt,name=counter" json:"counter,omitempty"`
	Summary          *Summary     `protobuf:"bytes,4,opt,name=summary" json:"summary,omitempty"`
	Untyped          *Untyped     `protobuf:"bytes,5,opt,name=untyped" json:"untyped,omitempty"`
	Histogram        *Histogram   `protobuf:"bytes,7,opt,name=histogram" json:"histogram,omitempty"`
	TimestampMs      *int64       `protobuf:"varint,6,opt,name=timestamp_ms" json:"timestamp_ms,omitempty"`
	XXX_unrecognized []byte       `json:"-"`
}

func (m *Metric) Reset()         { *m = Metric{} }
func (m *Metric) String() string { return proto.CompactTextString(m) }
func (*Metric) ProtoMessage()    {}

func (m *Metric) GetLabel() []*LabelPair {
	if m != nil {
		return m.Label
	}
	return nil
}

func (m *Metric) GetGauge() *Gauge {
	if m != nil {
		return m.Gauge
	}
	return nil
}

func (m *Metric) GetCounter() *Counter {
	if m != nil {
		return m.Counter
	}
	return nil
}

func (m *Metric) GetSummary() *Summary {
	if m != nil {
		return m.Summary
	}
	return nil
}

func (m *Metric) GetUntyped() *Untyped {
	if m != nil {
		return m.Untyped
	}
	return nil
}

func (m *Metric) GetHistogram() *Histogram {
	if m != nil {
		return m.Histogram
	}
	return nil
}

func (m *Metric) GetTimestampMs() int64 {
	if m != nil && m.TimestampMs != nil {
		return *m.TimestampMs
	}
	return 0
}

type MetricFamily struct {
	Name             *string     `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Help             *string     `protobuf:"bytes,2,opt,name=help" json:"help,omitempty"`
	Type             *MetricType `protobuf:"varint,3,opt,name=type,enum=io.prometheus.client.MetricType" json:"type,omitempty"`
	Metric           []*Metric   `protobuf:"bytes,4,rep,name=metric" json:"metric,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}

func (m *MetricFamily) Reset()         { *m = MetricFamily{} }
func (m *MetricFamily) String() string { return proto.CompactTextString(m) }
func (*MetricFamily) ProtoMessage()    {}

func (m *MetricFamily) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *MetricFamily) GetHelp() string {
	if m != nil && m.Help != nil {
		return *m.Help
	}
	return ""
}

func (m *MetricFamily) GetType() MetricType {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return MetricType_COUNTER
}

func (m *MetricFamily) GetMetric() []*Metric {
	if m != nil {
		return m.Metric
	}
	return nil
}

func init() {
	proto.RegisterEnum("io.prometheus.client.MetricType", MetricType_name, MetricType_value)
}
