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
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

func TestConcurrentSync(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		syncer  *fakeSyncer
		ids     []string
		want    []string
		wantErr string
	}{
		{
			name:   "no_errors",
			syncer: &fakeSyncer{},
			ids:    []string{"1", "2", "3"},
			want:   []string{"1", "2", "3"},
		},
		{
			name: "some_errors",
			syncer: &fakeSyncer{
				idErrs: map[string]error{
					"2": fmt.Errorf("sycer error"),
				},
			},
			ids:     []string{"1", "2", "3"},
			want:    []string{"1", "3"},
			wantErr: "failed to sync id 2",
		},
		{
			name: "all_errors",
			syncer: &fakeSyncer{
				idErrs: map[string]error{
					"1": fmt.Errorf("sycer error"),
					"2": fmt.Errorf("sycer error"),
					"3": fmt.Errorf("sycer error"),
				},
			},
			ids:     []string{"1", "2", "3"},
			wantErr: "failed to sync id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			err := ConcurrentSync(ctx, tc.syncer, tc.ids)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-got, +want) = %v", diff)
			}
			sort.Strings(tc.syncer.receivedIds)
			if diff := cmp.Diff(tc.syncer.receivedIds, tc.want); diff != "" {
				t.Errorf("unexpected result (-got, +want) = %v", diff)
			}
		})
	}
}

type fakeSyncer struct {
	receivedIds []string
	idErrs      map[string]error
	mutex       sync.Mutex
}

func (f *fakeSyncer) SourceSystem() string {
	return "testSource"
}

func (f *fakeSyncer) TargetSystem() string {
	return "testTarget"
}

func (f *fakeSyncer) Sync(_ context.Context, sourceGroupID string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if err, ok := f.idErrs[sourceGroupID]; ok {
		return err
	}
	f.receivedIds = append(f.receivedIds, sourceGroupID)
	return nil
}

func (f *fakeSyncer) SyncAll(ctx context.Context) error {
	panic("should not be called")
}
