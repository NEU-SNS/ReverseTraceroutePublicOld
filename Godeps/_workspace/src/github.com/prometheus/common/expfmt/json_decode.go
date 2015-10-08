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
// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"

	"github.com/prometheus/common/model"
)

type json2Decoder struct {
	dec  *json.Decoder
	fams []*dto.MetricFamily
}

func newJSON2Decoder(r io.Reader) Decoder {
	return &json2Decoder{
		dec: json.NewDecoder(r),
	}
}

type histogram002 struct {
	Labels model.LabelSet     `json:"labels"`
	Values map[string]float64 `json:"value"`
}

type counter002 struct {
	Labels model.LabelSet `json:"labels"`
	Value  float64        `json:"value"`
}

func protoLabelSet(base, ext model.LabelSet) []*dto.LabelPair {
	labels := base.Clone().Merge(ext)
	delete(labels, model.MetricNameLabel)

	names := make([]string, 0, len(labels))
	for ln := range labels {
		names = append(names, string(ln))
	}
	sort.Strings(names)

	pairs := make([]*dto.LabelPair, 0, len(labels))

	for _, ln := range names {
		lv := labels[model.LabelName(ln)]

		pairs = append(pairs, &dto.LabelPair{
			Name:  proto.String(ln),
			Value: proto.String(string(lv)),
		})
	}

	return pairs
}

func (d *json2Decoder) more() error {
	var entities []struct {
		BaseLabels model.LabelSet `json:"baseLabels"`
		Docstring  string         `json:"docstring"`
		Metric     struct {
			Type   string          `json:"type"`
			Values json.RawMessage `json:"value"`
		} `json:"metric"`
	}

	if err := d.dec.Decode(&entities); err != nil {
		return err
	}
	for _, e := range entities {
		f := &dto.MetricFamily{
			Name:   proto.String(string(e.BaseLabels[model.MetricNameLabel])),
			Help:   proto.String(e.Docstring),
			Type:   dto.MetricType_UNTYPED.Enum(),
			Metric: []*dto.Metric{},
		}

		d.fams = append(d.fams, f)

		switch e.Metric.Type {
		case "counter", "gauge":
			var values []counter002

			if err := json.Unmarshal(e.Metric.Values, &values); err != nil {
				return fmt.Errorf("could not extract %s value: %s", e.Metric.Type, err)
			}

			for _, ctr := range values {
				f.Metric = append(f.Metric, &dto.Metric{
					Label: protoLabelSet(e.BaseLabels, ctr.Labels),
					Untyped: &dto.Untyped{
						Value: proto.Float64(ctr.Value),
					},
				})
			}

		case "histogram":
			var values []histogram002

			if err := json.Unmarshal(e.Metric.Values, &values); err != nil {
				return fmt.Errorf("could not extract %s value: %s", e.Metric.Type, err)
			}

			for _, hist := range values {
				quants := make([]string, 0, len(values))
				for q := range hist.Values {
					quants = append(quants, q)
				}

				sort.Strings(quants)

				for _, q := range quants {
					value := hist.Values[q]
					// The correct label is "quantile" but to not break old expressions
					// this remains "percentile"
					hist.Labels["percentile"] = model.LabelValue(q)

					f.Metric = append(f.Metric, &dto.Metric{
						Label: protoLabelSet(e.BaseLabels, hist.Labels),
						Untyped: &dto.Untyped{
							Value: proto.Float64(value),
						},
					})
				}
			}

		default:
			return fmt.Errorf("unknown metric type %q", e.Metric.Type)
		}
	}
	return nil
}

// Decode implements the Decoder interface.
func (d *json2Decoder) Decode(v *dto.MetricFamily) error {
	if len(d.fams) == 0 {
		if err := d.more(); err != nil {
			return err
		}
	}

	*v = *d.fams[0]
	d.fams = d.fams[1:]

	return nil
}
