/*
Copyright 2014 The Kubernetes Authors.

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

package record

//
//import (
//	"encoding/json"
//	"fmt"
//	"net/http"
//	"strconv"
//	"sync"
//	"testing"
//	"time"
//
//	"github.com/yubo/apiserver/pkg/rest"
//	"github.com/yubo/apiserver/pkg/util/strategicpatch"
//	ref "github.com/yubo/apiserver/tools/reference"
//	"github.com/yubo/golib/api"
//	"github.com/yubo/golib/api/errors"
//	k8sruntime "github.com/yubo/golib/pkg/runtime"
//	"github.com/yubo/golib/util/clock"
//)
//
//type testEventSink struct {
//	OnCreate func(e *api.Event) (*api.Event, error)
//	OnUpdate func(e *api.Event) (*api.Event, error)
//	OnPatch  func(e *api.Event, p []byte) (*api.Event, error)
//}
//
//// CreateEvent records the event for testing.
//func (t *testEventSink) Create(e *api.Event) (*api.Event, error) {
//	if t.OnCreate != nil {
//		return t.OnCreate(e)
//	}
//	return e, nil
//}
//
//// UpdateEvent records the event for testing.
//func (t *testEventSink) Update(e *api.Event) (*api.Event, error) {
//	if t.OnUpdate != nil {
//		return t.OnUpdate(e)
//	}
//	return e, nil
//}
//
//// PatchEvent records the event for testing.
//func (t *testEventSink) Patch(e *api.Event, p []byte) (*api.Event, error) {
//	if t.OnPatch != nil {
//		return t.OnPatch(e, p)
//	}
//	return e, nil
//}
//
//type OnCreateFunc func(*api.Event) (*api.Event, error)
//
//func OnCreateFactory(testCache map[string]*api.Event, createEvent chan<- *api.Event) OnCreateFunc {
//	return func(event *api.Event) (*api.Event, error) {
//		testCache[getEventKey(event)] = event
//		createEvent <- event
//		return event, nil
//	}
//}
//
//type OnPatchFunc func(*api.Event, []byte) (*api.Event, error)
//
//func OnPatchFactory(testCache map[string]*api.Event, patchEvent chan<- *api.Event) OnPatchFunc {
//	return func(event *api.Event, patch []byte) (*api.Event, error) {
//		cachedEvent, found := testCache[getEventKey(event)]
//		if !found {
//			return nil, fmt.Errorf("unexpected error: couldn't find Event in testCache.")
//		}
//		originalData, err := json.Marshal(cachedEvent)
//		if err != nil {
//			return nil, fmt.Errorf("unexpected error: %v", err)
//		}
//		patched, err := strategicpatch.StrategicMergePatch(originalData, patch, event)
//		if err != nil {
//			return nil, fmt.Errorf("unexpected error: %v", err)
//		}
//		patchedObj := &api.Event{}
//		err = json.Unmarshal(patched, patchedObj)
//		if err != nil {
//			return nil, fmt.Errorf("unexpected error: %v", err)
//		}
//		patchEvent <- patchedObj
//		return patchedObj, nil
//	}
//}
//
//func TestNonRacyShutdown(t *testing.T) {
//	// Attempt to simulate previously racy conditions, and ensure that no race
//	// occurs: Nominally, calling "Eventf" *followed by* shutdown from the same
//	// thread should be a safe operation, but it's not if we launch recorder.Action
//	// in a goroutine.
//
//	caster := NewBroadcasterForTests(0)
//	clock := clock.NewFakeClock(time.Now())
//	recorder := recorderWithFakeClock(api.EventSource{Component: "eventTest"}, caster, clock)
//
//	var wg sync.WaitGroup
//	wg.Add(100)
//	for i := 0; i < 100; i++ {
//		go func() {
//			defer wg.Done()
//			recorder.Eventf(&api.ObjectReference{}, api.EventTypeNormal, "Started", "blah")
//		}()
//	}
//
//	wg.Wait()
//	caster.Shutdown()
//}
//
//func TestEventf(t *testing.T) {
//	testPod := &api.Pod{
//		ObjectMeta: api.ObjectMeta{
//			SelfLink:  "/api/v1/namespaces/baz/pods/foo",
//			Name:      "foo",
//			Namespace: "baz",
//			UID:       "bar",
//		},
//	}
//	testPod2 := &api.Pod{
//		ObjectMeta: api.ObjectMeta{
//			SelfLink:  "/api/v1/namespaces/baz/pods/foo",
//			Name:      "foo",
//			Namespace: "baz",
//			UID:       "differentUid",
//		},
//	}
//	testRef, err := ref.GetPartialReference(testPod, "spec.containers[2]")
//	if err != nil {
//		t.Fatal(err)
//	}
//	testRef2, err := ref.GetPartialReference(testPod2, "spec.containers[3]")
//	if err != nil {
//		t.Fatal(err)
//	}
//	table := []struct {
//		obj          k8sruntime.Object
//		eventtype    string
//		reason       string
//		messageFmt   string
//		elements     []interface{}
//		expect       *api.Event
//		expectLog    string
//		expectUpdate bool
//	}{
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testPod,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Killed",
//			messageFmt: "some other verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//				},
//				Reason:  "Killed",
//				Message: "some other verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:""}): type: 'Normal' reason: 'Killed' some other verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   2,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: true,
//		},
//		{
//			obj:        testRef2,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "differentUid",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[3]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"differentUid", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[3]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   3,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: true,
//		},
//		{
//			obj:        testRef2,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Stopped",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "differentUid",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[3]",
//				},
//				Reason:  "Stopped",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"differentUid", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[3]"}): type: 'Normal' reason: 'Stopped' some verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testRef2,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Stopped",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "differentUid",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[3]",
//				},
//				Reason:  "Stopped",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   2,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"differentUid", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[3]"}): type: 'Normal' reason: 'Stopped' some verbose message: 1`,
//			expectUpdate: true,
//		},
//	}
//
//	testCache := map[string]*api.Event{}
//	logCalled := make(chan struct{})
//	createEvent := make(chan *api.Event)
//	updateEvent := make(chan *api.Event)
//	patchEvent := make(chan *api.Event)
//	testEvents := testEventSink{
//		OnCreate: OnCreateFactory(testCache, createEvent),
//		OnUpdate: func(event *api.Event) (*api.Event, error) {
//			updateEvent <- event
//			return event, nil
//		},
//		OnPatch: OnPatchFactory(testCache, patchEvent),
//	}
//	eventBroadcaster := NewBroadcasterForTests(0)
//	sinkWatcher := eventBroadcaster.StartRecordingToSink(&testEvents)
//
//	clock := clock.NewFakeClock(time.Now())
//	recorder := recorderWithFakeClock(api.EventSource{Component: "eventTest"}, eventBroadcaster, clock)
//	for index, item := range table {
//		clock.Step(1 * time.Second)
//		logWatcher := eventBroadcaster.StartLogging(func(formatter string, args ...interface{}) {
//			if e, a := item.expectLog, fmt.Sprintf(formatter, args...); e != a {
//				t.Errorf("Expected '%v', got '%v'", e, a)
//			}
//			logCalled <- struct{}{}
//		})
//		recorder.Eventf(item.obj, item.eventtype, item.reason, item.messageFmt, item.elements...)
//
//		<-logCalled
//
//		// validate event
//		if item.expectUpdate {
//			actualEvent := <-patchEvent
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		} else {
//			actualEvent := <-createEvent
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		}
//		logWatcher.Stop()
//	}
//	sinkWatcher.Stop()
//}
//
//func recorderWithFakeClock(eventSource api.EventSource, eventBroadcaster EventBroadcaster, clock clock.Clock) EventRecorder {
//	return &recorderImpl{eventSource, eventBroadcaster.(*eventBroadcasterImpl).Broadcaster, clock}
//}
//
//func TestWriteEventError(t *testing.T) {
//	type entry struct {
//		timesToSendError int
//		attemptsWanted   int
//		err              error
//	}
//	table := map[string]*entry{
//		"giveUp1": {
//			timesToSendError: 1000,
//			attemptsWanted:   1,
//			err:              &rest.RequestConstructionError{},
//		},
//		"giveUp2": {
//			timesToSendError: 1000,
//			attemptsWanted:   1,
//			err:              &errors.StatusError{},
//		},
//		"retry1": {
//			timesToSendError: 1000,
//			attemptsWanted:   12,
//			err:              &errors.UnexpectedObjectError{},
//		},
//		"retry2": {
//			timesToSendError: 1000,
//			attemptsWanted:   12,
//			err:              fmt.Errorf("A weird error"),
//		},
//		"succeedEventually": {
//			timesToSendError: 2,
//			attemptsWanted:   2,
//			err:              fmt.Errorf("A weird error"),
//		},
//	}
//
//	clock := clock.IntervalClock{Time: time.Now(), Duration: time.Second}
//	eventCorrelator := NewEventCorrelator(&clock)
//
//	for caseName, ent := range table {
//		attempts := 0
//		sink := &testEventSink{
//			OnCreate: func(event *api.Event) (*api.Event, error) {
//				attempts++
//				if attempts < ent.timesToSendError {
//					return nil, ent.err
//				}
//				return event, nil
//			},
//		}
//		ev := &api.Event{}
//		recordToSink(sink, ev, eventCorrelator, 0)
//		if attempts != ent.attemptsWanted {
//			t.Errorf("case %v: wanted %d, got %d attempts", caseName, ent.attemptsWanted, attempts)
//		}
//	}
//}
//
//func TestUpdateExpiredEvent(t *testing.T) {
//	clock := clock.IntervalClock{Time: time.Now(), Duration: time.Second}
//	eventCorrelator := NewEventCorrelator(&clock)
//
//	var createdEvent *api.Event
//
//	sink := &testEventSink{
//		OnPatch: func(*api.Event, []byte) (*api.Event, error) {
//			return nil, &errors.StatusError{
//				ErrStatus: api.Status{
//					Code:   http.StatusNotFound,
//					Reason: api.StatusReasonNotFound,
//				}}
//		},
//		OnCreate: func(event *api.Event) (*api.Event, error) {
//			createdEvent = event
//			return event, nil
//		},
//	}
//
//	ev := &api.Event{}
//	ev.ResourceVersion = "updated-resource-version"
//	ev.Count = 2
//	recordToSink(sink, ev, eventCorrelator, 0)
//
//	if createdEvent == nil {
//		t.Error("Event did not get created after patch failed")
//		return
//	}
//
//	if createdEvent.ResourceVersion != "" {
//		t.Errorf("Event did not have its resource version cleared, was %s", createdEvent.ResourceVersion)
//	}
//}
//
//func TestLotsOfEvents(t *testing.T) {
//	recorderCalled := make(chan struct{})
//	loggerCalled := make(chan struct{})
//
//	// Fail each event a few times to ensure there's some load on the tested code.
//	var counts [1000]int
//	testEvents := testEventSink{
//		OnCreate: func(event *api.Event) (*api.Event, error) {
//			num, err := strconv.Atoi(event.Message)
//			if err != nil {
//				t.Error(err)
//				return event, nil
//			}
//			counts[num]++
//			if counts[num] < 5 {
//				return nil, fmt.Errorf("fake error")
//			}
//			recorderCalled <- struct{}{}
//			return event, nil
//		},
//	}
//
//	eventBroadcaster := NewBroadcasterForTests(0)
//	sinkWatcher := eventBroadcaster.StartRecordingToSink(&testEvents)
//	logWatcher := eventBroadcaster.StartLogging(func(formatter string, args ...interface{}) {
//		loggerCalled <- struct{}{}
//	})
//	recorder := eventBroadcaster.NewRecorder(api.EventSource{Component: "eventTest"})
//	for i := 0; i < maxQueuedEvents; i++ {
//		// we want a unique object to stop spam filtering
//		ref := &api.ObjectReference{
//			Kind:       "Pod",
//			Name:       fmt.Sprintf("foo-%v", i),
//			Namespace:  "baz",
//			UID:        "bar",
//			APIVersion: "version",
//		}
//		// we need to vary the reason to prevent aggregation
//		go recorder.Eventf(ref, api.EventTypeNormal, "Reason-"+strconv.Itoa(i), strconv.Itoa(i))
//	}
//	// Make sure no events were dropped by either of the listeners.
//	for i := 0; i < maxQueuedEvents; i++ {
//		<-recorderCalled
//		<-loggerCalled
//	}
//	// Make sure that every event was attempted 5 times
//	for i := 0; i < maxQueuedEvents; i++ {
//		if counts[i] < 5 {
//			t.Errorf("Only attempted to record event '%d' %d times.", i, counts[i])
//		}
//	}
//	sinkWatcher.Stop()
//	logWatcher.Stop()
//}
//
//func TestEventfNoNamespace(t *testing.T) {
//	testPod := &api.Pod{
//		ObjectMeta: api.ObjectMeta{
//			SelfLink: "/api/v1/namespaces/default/pods/foo",
//			Name:     "foo",
//			UID:      "bar",
//		},
//	}
//	testRef, err := ref.GetPartialReference(testPod, "spec.containers[2]")
//	if err != nil {
//		t.Fatal(err)
//	}
//	table := []struct {
//		obj          k8sruntime.Object
//		eventtype    string
//		reason       string
//		messageFmt   string
//		elements     []interface{}
//		expect       *api.Event
//		expectLog    string
//		expectUpdate bool
//	}{
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "default",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: false,
//		},
//	}
//
//	testCache := map[string]*api.Event{}
//	logCalled := make(chan struct{})
//	createEvent := make(chan *api.Event)
//	updateEvent := make(chan *api.Event)
//	patchEvent := make(chan *api.Event)
//	testEvents := testEventSink{
//		OnCreate: OnCreateFactory(testCache, createEvent),
//		OnUpdate: func(event *api.Event) (*api.Event, error) {
//			updateEvent <- event
//			return event, nil
//		},
//		OnPatch: OnPatchFactory(testCache, patchEvent),
//	}
//	eventBroadcaster := NewBroadcasterForTests(0)
//	sinkWatcher := eventBroadcaster.StartRecordingToSink(&testEvents)
//
//	clock := clock.NewFakeClock(time.Now())
//	recorder := recorderWithFakeClock(api.EventSource{Component: "eventTest"}, eventBroadcaster, clock)
//
//	for index, item := range table {
//		clock.Step(1 * time.Second)
//		logWatcher := eventBroadcaster.StartLogging(func(formatter string, args ...interface{}) {
//			if e, a := item.expectLog, fmt.Sprintf(formatter, args...); e != a {
//				t.Errorf("Expected '%v', got '%v'", e, a)
//			}
//			logCalled <- struct{}{}
//		})
//		recorder.Eventf(item.obj, item.eventtype, item.reason, item.messageFmt, item.elements...)
//
//		<-logCalled
//
//		// validate event
//		if item.expectUpdate {
//			actualEvent := <-patchEvent
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		} else {
//			actualEvent := <-createEvent
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		}
//
//		logWatcher.Stop()
//	}
//	sinkWatcher.Stop()
//}
//
//func TestMultiSinkCache(t *testing.T) {
//	testPod := &api.Pod{
//		ObjectMeta: api.ObjectMeta{
//			SelfLink:  "/api/v1/namespaces/baz/pods/foo",
//			Name:      "foo",
//			Namespace: "baz",
//			UID:       "bar",
//		},
//	}
//	testPod2 := &api.Pod{
//		ObjectMeta: api.ObjectMeta{
//			SelfLink:  "/api/v1/namespaces/baz/pods/foo",
//			Name:      "foo",
//			Namespace: "baz",
//			UID:       "differentUid",
//		},
//	}
//	testRef, err := ref.GetPartialReference(testPod, "spec.containers[2]")
//	if err != nil {
//		t.Fatal(err)
//	}
//	testRef2, err := ref.GetPartialReference(testPod2, "spec.containers[3]")
//	if err != nil {
//		t.Fatal(err)
//	}
//	table := []struct {
//		obj          k8sruntime.Object
//		eventtype    string
//		reason       string
//		messageFmt   string
//		elements     []interface{}
//		expect       *api.Event
//		expectLog    string
//		expectUpdate bool
//	}{
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testPod,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Killed",
//			messageFmt: "some other verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//				},
//				Reason:  "Killed",
//				Message: "some other verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:""}): type: 'Normal' reason: 'Killed' some other verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   2,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: true,
//		},
//		{
//			obj:        testRef2,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "differentUid",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[3]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"differentUid", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[3]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testRef,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Started",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "bar",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[2]",
//				},
//				Reason:  "Started",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   3,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"bar", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[2]"}): type: 'Normal' reason: 'Started' some verbose message: 1`,
//			expectUpdate: true,
//		},
//		{
//			obj:        testRef2,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Stopped",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "differentUid",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[3]",
//				},
//				Reason:  "Stopped",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   1,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"differentUid", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[3]"}): type: 'Normal' reason: 'Stopped' some verbose message: 1`,
//			expectUpdate: false,
//		},
//		{
//			obj:        testRef2,
//			eventtype:  api.EventTypeNormal,
//			reason:     "Stopped",
//			messageFmt: "some verbose message: %v",
//			elements:   []interface{}{1},
//			expect: &api.Event{
//				ObjectMeta: api.ObjectMeta{
//					Name:      "foo",
//					Namespace: "baz",
//				},
//				InvolvedObject: api.ObjectReference{
//					Kind:       "Pod",
//					Name:       "foo",
//					Namespace:  "baz",
//					UID:        "differentUid",
//					APIVersion: "v1",
//					FieldPath:  "spec.containers[3]",
//				},
//				Reason:  "Stopped",
//				Message: "some verbose message: 1",
//				Source:  api.EventSource{Component: "eventTest"},
//				Count:   2,
//				Type:    api.EventTypeNormal,
//			},
//			expectLog:    `Event(api.ObjectReference{Kind:"Pod", Namespace:"baz", Name:"foo", UID:"differentUid", APIVersion:"v1", ResourceVersion:"", FieldPath:"spec.containers[3]"}): type: 'Normal' reason: 'Stopped' some verbose message: 1`,
//			expectUpdate: true,
//		},
//	}
//
//	testCache := map[string]*api.Event{}
//	createEvent := make(chan *api.Event)
//	updateEvent := make(chan *api.Event)
//	patchEvent := make(chan *api.Event)
//	testEvents := testEventSink{
//		OnCreate: OnCreateFactory(testCache, createEvent),
//		OnUpdate: func(event *api.Event) (*api.Event, error) {
//			updateEvent <- event
//			return event, nil
//		},
//		OnPatch: OnPatchFactory(testCache, patchEvent),
//	}
//
//	testCache2 := map[string]*api.Event{}
//	createEvent2 := make(chan *api.Event)
//	updateEvent2 := make(chan *api.Event)
//	patchEvent2 := make(chan *api.Event)
//	testEvents2 := testEventSink{
//		OnCreate: OnCreateFactory(testCache2, createEvent2),
//		OnUpdate: func(event *api.Event) (*api.Event, error) {
//			updateEvent2 <- event
//			return event, nil
//		},
//		OnPatch: OnPatchFactory(testCache2, patchEvent2),
//	}
//
//	eventBroadcaster := NewBroadcasterForTests(0)
//	clock := clock.NewFakeClock(time.Now())
//	recorder := recorderWithFakeClock(api.EventSource{Component: "eventTest"}, eventBroadcaster, clock)
//
//	sinkWatcher := eventBroadcaster.StartRecordingToSink(&testEvents)
//	for index, item := range table {
//		clock.Step(1 * time.Second)
//		recorder.Eventf(item.obj, item.eventtype, item.reason, item.messageFmt, item.elements...)
//
//		// validate event
//		if item.expectUpdate {
//			actualEvent := <-patchEvent
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		} else {
//			actualEvent := <-createEvent
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		}
//	}
//
//	// Another StartRecordingToSink call should start to record events with new clean cache.
//	sinkWatcher2 := eventBroadcaster.StartRecordingToSink(&testEvents2)
//	for index, item := range table {
//		clock.Step(1 * time.Second)
//		recorder.Eventf(item.obj, item.eventtype, item.reason, item.messageFmt, item.elements...)
//
//		// validate event
//		if item.expectUpdate {
//			actualEvent := <-patchEvent2
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		} else {
//			actualEvent := <-createEvent2
//			validateEvent(strconv.Itoa(index), actualEvent, item.expect, t)
//		}
//	}
//
//	sinkWatcher.Stop()
//	sinkWatcher2.Stop()
//}
