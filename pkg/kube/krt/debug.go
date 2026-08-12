// Copyright Istio Authors
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

package krt

import (
	"encoding/json"
	"sync"

	"istio.io/istio/pkg/slices"
)

// DebugHandler allows attaching a variety of collections to it and then dumping them
type DebugHandler struct {
	debugCollections []DebugCollection
	mu               sync.RWMutex
}

func (p *DebugHandler) MarshalJSON() ([]byte, error) {
	p.mu.Lock()
	p.pruneStopped()
	collections := slices.Clone(p.debugCollections)
	p.mu.Unlock()
	return json.Marshal(collections)
}

// pruneStopped drops collections that have been stopped. Callers must hold the write lock.
// Without this, the debugger holds a reference to every collection ever created; in a multicluster setup
// collections are recreated each time a remote cluster is rebuilt (for instance on a token rotation), so the
// list -- and everything it keeps alive -- grows without bound.
func (p *DebugHandler) pruneStopped() {
	p.debugCollections = slices.FilterInPlace(p.debugCollections, func(c DebugCollection) bool {
		return !c.stopped()
	})
}

var GlobalDebugHandler = new(DebugHandler)

type CollectionDump struct {
	// Map of output key -> output
	Outputs map[string]any `json:"outputs,omitempty"`
	// Name of the input collection
	InputCollection string `json:"inputCollection,omitempty"`
	// Map of input key -> info
	Inputs map[string]InputDump `json:"inputs,omitempty"`
	// Synced returns whether the collection is synced or not
	Synced bool `json:"synced"`
}
type InputDump struct {
	Outputs      []string `json:"outputs,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}
type DebugCollection struct {
	name string
	dump func() CollectionDump
	uid  collectionUID
	// stop is closed when the collection is no longer running, at which point it can be dropped from the debugger.
	stop <-chan struct{}
}

func (p DebugCollection) stopped() bool {
	if p.stop == nil {
		return false
	}
	select {
	case <-p.stop:
		return true
	default:
		return false
	}
}

func (p DebugCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"uid":   p.uid,
		"name":  p.name,
		"state": p.dump(),
	})
}

// maybeRegisterCollectionForDebugging registers the collection in the debugger, if one is enabled.
// stop is the collection's stop channel; the entry is dropped once it is closed.
func maybeRegisterCollectionForDebugging[T any](c Collection[T], handler *DebugHandler, stop <-chan struct{}) {
	if handler == nil {
		return
	}
	cc := c.(internalCollection[T])
	handler.mu.Lock()
	defer handler.mu.Unlock()
	handler.pruneStopped()
	handler.debugCollections = append(handler.debugCollections, DebugCollection{
		name: cc.name(),
		dump: cc.dump,
		uid:  cc.uid(),
		stop: stop,
	})
	log.Debugf("XXXX [KRT-DEBUG] (+) collection registered in DebugHandler name=%v uid=%v debugHandlerEntries=%d totalCollectionsCreated=%d",
		cc.name(), cc.uid(), len(handler.debugCollections), CollectionsCreated())
}

// nolint: unused // (not true, not sure why it thinks it is!)
func eraseMap[T any](l map[Key[T]]T) map[string]any {
	nm := make(map[string]any, len(l))
	for k, v := range l {
		nm[string(k)] = v
	}
	return nm
}
