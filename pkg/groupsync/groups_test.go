// Copyright 2025 Google LLC
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
	"testing"

	"github.com/abcxyz/pkg/testutil"
)

func TestNoopUserMapper_MappedUserID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		inputID string
		wantID  string
		wantErr string
	}{
		{
			name:    "normal_case",
			inputID: "test_user_123",
			wantID:  "test_user_123",
		},
		{
			name:    "empty_string",
			inputID: "",
			wantID:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			mapper := NewNoopUserMapper()
			gotID, gotErr := mapper.MappedUserID(ctx, tc.inputID)
			if diff := testutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("unexpected error (-want, +got):\n%s", diff)
			}
			if gotID != tc.wantID {
				t.Errorf("MappedUserID() gotID = %v, wantID %v", gotID, tc.wantID)
			}
		})
	}
}
