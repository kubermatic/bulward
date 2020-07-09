/*
Copyright 2020 The Bulward Authors.

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

package events

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/watch"
)

func NewTracer(watch watch.Interface, invariants ...Predicate) *Tracer {
	return &Tracer{
		watch:      watch,
		invariants: invariants,
	}
}

type Tracer struct {
	watch      watch.Interface
	invariants []Predicate
}

func (wt *Tracer) WaitUntil(ctx context.Context, predicate Predicate) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-wt.watch.ResultChan():
			if !ok {
				return io.EOF
			}
			if err := wt.checkInvariants(ev); err != nil {
				return err
			}
			done, err := predicate(ev)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

func (wt *Tracer) checkInvariants(ev watch.Event) error {
	for _, inv := range wt.invariants {
		ok, err := inv(ev)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("invariant broken\n%v", ev)
		}
	}
	return nil
}

func (wt *Tracer) Stop() error {
	wt.watch.Stop()
	for ev := range wt.watch.ResultChan() {
		if err := wt.checkInvariants(ev); err != nil {
			return err
		}
	}
	return nil
}

func (wt *Tracer) TestCleanupFunc(t *testing.T) func() {
	return func() {
		require.NoError(t, wt.Stop())
	}
}
