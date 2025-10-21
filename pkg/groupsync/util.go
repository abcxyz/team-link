// Copyright 2024 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package groupsync

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/abcxyz/team-link/v2/apis/v1alpha3"
)

// ConcurrentSync syncs the given source groups concurrently using the given syncer.
// The level of concurrency is based of the value of runtime.NumCPU.
func ConcurrentSync(ctx context.Context, syncer v1alpha3.GroupSyncer, sourceGroupIDs []string) error {
	return concurrentSyncFunc(ctx, sourceGroupIDs, syncer.Sync)
}

func concurrentSyncFunc(ctx context.Context, groupIDs []string, syncFn func(context.Context, string) error) error {
	ids := make(chan string, len(groupIDs))
	errs := make(chan error, len(groupIDs))
	for _, id := range groupIDs {
		ids <- id
	}
	close(ids)
	waitGroup := sync.WaitGroup{}
	for i := 0; i < runtime.NumCPU(); i++ {
		waitGroup.Go(func() {
			for id := range ids {
				if err := syncFn(ctx, id); err != nil {
					errs <- fmt.Errorf("failed to sync id %s: %w", id, err)
				}
			}
		})
	}
	waitGroup.Wait()
	close(errs)
	var merr error
	for e := range errs {
		merr = errors.Join(merr, e)
	}
	return merr
}
