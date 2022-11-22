// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/servicegraphprocessor/internal/store"

import (
	"container/list"
	"errors"
	semconv "go.opentelemetry.io/collector/semconv/v1.13.0"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

var (
	ErrTooManyItems      = errors.New("too many items")
	NeedToFindAttributes = []string{semconv.AttributeNetSockHostAddr, semconv.AttributeRPCService, semconv.AttributeHTTPURL, semconv.AttributeHTTPTarget, semconv.AttributeHTTPURL, semconv.AttributeNetPeerName, semconv.AttributeNetHostName}
)

type Callback func(e *Edge)

type Key struct {
	tid pcommon.TraceID
	sid pcommon.SpanID
}

func NewKey(tid pcommon.TraceID, sid pcommon.SpanID) Key {
	return Key{tid: tid, sid: sid}
}

type Store struct {
	l   *list.List
	mtx sync.Mutex
	m   map[Key]*list.Element

	onComplete Callback
	onExpire   Callback

	ttl      time.Duration
	maxItems int
}

// NewStore creates a Store to build service graphs. The store caches edges, each representing a
// request between two services. Once an edge is complete its metrics can be collected. Edges that
// have not found their pair are deleted after ttl time.
func NewStore(ttl time.Duration, maxItems int, onComplete, onExpire Callback) *Store {
	s := &Store{
		l: list.New(),
		m: make(map[Key]*list.Element),

		onComplete: onComplete,
		onExpire:   onExpire,

		ttl:      ttl,
		maxItems: maxItems,
	}

	return s
}

// len is only used for testing.
func (s *Store) len() int {
	return s.l.Len()
}

// UpsertEdge fetches an Edge from the store and updates it using the given callback. If the Edge
// doesn't exist yet, it creates a new one with the default TTL.
// If the Edge is complete after applying the callback, it's completed and removed.
func (s *Store) UpsertEdge(key Key, update Callback) (isNew bool, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if storedEdge, ok := s.m[key]; ok {
		edge := storedEdge.Value.(*Edge)
		update(edge)

		if edge.isComplete() {
			s.onComplete(edge)
			delete(s.m, key)
			s.l.Remove(storedEdge)
		}

		return false, nil
	}

	edge := newEdge(key, s.ttl)
	update(edge)

	if edge.isComplete() {
		s.onComplete(edge)
		return true, nil
	}

	// Check we can add new edges
	if s.l.Len() >= s.maxItems {
		// TODO: try to evict expired items
		return false, ErrTooManyItems
	}

	ele := s.l.PushBack(edge)
	s.m[key] = ele

	return true, nil
}

// Expire evicts all expired items in the store.
func (s *Store) Expire() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Iterates until no more items can be evicted
	for s.trySpeculateEvictHead() {
		s.tryEvictHead()
	}
}

// speculate virtual node before edge get expired.
func (s *Store) trySpeculateEvictHead() bool {
	head := s.l.Front()
	if head == nil {
		return false // list is empty
	}
	headEdge := head.Value.(*Edge)
	if !headEdge.isExpired() {
		return false
	}

	if len(headEdge.ClientService) == 0 {
		headEdge.ClientService = "user"
	}

	if len(headEdge.ServerService) == 0 {
		headEdge.ServerService = s.getPeerHost(NeedToFindAttributes, headEdge.Peer.RpcAttributes)
	}

	if headEdge.isComplete() {
		s.onComplete(headEdge)
	}
	return true
}

func (s *Store) getPeerHost(m []string, peers map[string]string) string {
	peerStr := "unknown"
	for _, s := range m {
		if len(peers[s]) != 0 {
			peerStr = peers[s]
		}
	}
	return peerStr
}

// tryEvictHead checks if the oldest item (head of list) can be evicted and will delete it if so.
// Returns true if the head was evicted.
//
// Must be called holding lock.
func (s *Store) tryEvictHead() bool {
	head := s.l.Front()
	if head == nil {
		return false // list is empty
	}

	headEdge := head.Value.(*Edge)
	if !headEdge.isExpired() {
		return false
	}

	s.onExpire(headEdge)
	delete(s.m, headEdge.key)
	s.l.Remove(head)

	return true
}
