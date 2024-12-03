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

package client

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/abcxyz/pkg/testutil"
	tltypes "github.com/abcxyz/team-link/internal"
)

func TestCreateGoogleGroupToGithubMapper(t *testing.T) {
	t.Parallel()
	defaultWritePath := "test.textproto"
	cases := []struct {
		name                          string
		fileReadpath                  string
		content                       string
		wantGoogleGroupToGitHubMapper *GoogleGroupToGitHubMapper
		wantErr                       string
	}{
		{
			name: "success",
			content: `
mappings: [
  {
    google_group: {
      group_id: "test_id_1"
    }
    git_hub_team: {
      org_id: 1
      team_id: 2
    }
  },
  {
    google_group: {
      group_id: "test_id_1"
    }
    git_hub_team: {
      org_id: 1
      team_id: 3
    }
  },
  {
    google_group: {
      group_id: "test_id_2"
    }
    git_hub_team: {
      org_id: 1
      team_id: 4
    }
  }
]
`,
			wantGoogleGroupToGitHubMapper: &GoogleGroupToGitHubMapper{
				GoogleGroupToGitHubTeam: map[string][]string{
					"test_id_1": {"1:2", "1:3"},
					"test_id_2": {"1:4"},
				},
			},
		},
		{
			name:         "file_not_exist",
			fileReadpath: "not_exist_path",
			wantErr:      "failed to read mapping file",
		},
		{
			name:    "invalid_format",
			content: `not valid`,
			wantErr: "failed to unmarshal mapping file",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// ctx := context.Background()
			td := t.TempDir()

			// Create a file in the temporary directory
			tempFile, err := os.CreateTemp(td, defaultWritePath)
			if err != nil {
				t.Fatal("failed to create tempFile: %w", err)
			}
			defer os.Remove(tempFile.Name())

			// Write some data to the file
			_, err = tempFile.WriteString(tc.content)
			if err != nil {
				t.Fatal("failed to write tempFile: %w", err)
			}

			if tc.fileReadpath == "" {
				tc.fileReadpath = tempFile.Name()
			}
			res, err := NewOneToManyGroupMapper(tltypes.SystemTypeGoogleGroups, tltypes.SystemTypeGitHub, tc.fileReadpath)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(res, tc.wantGoogleGroupToGitHubMapper, protocmp.Transform()); diff != "" {
				t.Errorf("got unexpected response:\n%s", diff)
			}
		})
	}
}
