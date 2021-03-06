/*
Copyright 2020 The Kubernetes Authors.

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

package filters

import (
	"context"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/metrics"
	utilclock "github.com/yubo/golib/util/clock"
	apirequest "github.com/yubo/apiserver/pkg/request"
)

type requestFilterRecordKeyType int

// requestFilterRecordKey is the context key for a request filter record struct.
const requestFilterRecordKey requestFilterRecordKeyType = iota

type requestFilterRecord struct {
	name             string
	startedTimestamp time.Time
}

// withRequestFilterRecord attaches the given request filter record to the parent context.
func withRequestFilterRecord(parent context.Context, fr *requestFilterRecord) context.Context {
	return apirequest.WithValue(parent, requestFilterRecordKey, fr)
}

// requestFilterRecordFrom returns the request filter record from the given context.
func requestFilterRecordFrom(ctx context.Context) *requestFilterRecord {
	fr, _ := ctx.Value(requestFilterRecordKey).(*requestFilterRecord)
	return fr
}

// TrackStarted measures the timestamp the given handler has started execution
// by attaching a handler to the chain.
func TrackStarted(handler http.Handler, name string) http.Handler {
	return trackStarted(handler, name, utilclock.RealClock{})
}

// TrackCompleted measures the timestamp the given handler has completed execution and then
// it updates the corresponding metric with the filter latency duration.
func TrackCompleted(handler http.Handler) http.Handler {
	return trackCompleted(handler, utilclock.RealClock{}, func(ctx context.Context, fr *requestFilterRecord, completedAt time.Time) {
		metrics.RecordFilterLatency(ctx, fr.name, completedAt.Sub(fr.startedTimestamp))
	})
}

func trackStarted(handler http.Handler, name string, clock utilclock.PassiveClock) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if fr := requestFilterRecordFrom(ctx); fr != nil {
			fr.name = name
			fr.startedTimestamp = clock.Now()

			handler.ServeHTTP(w, r)
			return
		}

		fr := &requestFilterRecord{
			name:             name,
			startedTimestamp: clock.Now(),
		}
		r = r.WithContext(withRequestFilterRecord(ctx, fr))
		handler.ServeHTTP(w, r)
	})
}

func trackCompleted(handler http.Handler, clock utilclock.PassiveClock, action func(context.Context, *requestFilterRecord, time.Time)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The previous filter has just completed.
		completedAt := clock.Now()

		defer handler.ServeHTTP(w, r)

		ctx := r.Context()
		if fr := requestFilterRecordFrom(ctx); fr != nil {
			action(ctx, fr, completedAt)
		}
	})
}
