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

package googlegroupgithub

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/testutil"
)

func TestCreateBidirectionalGoogleGroupGitHubMapper(t *testing.T) {
	t.Parallel()
	defaultWritePath := "test.textproto"
	cases := []struct {
		name                          string
		fileReadpath                  string
		content                       string
		wantGoogleGroupToGitHubMapper *GroupMapper
		wantGitHubToGoogleGroupMapper *GroupMapper
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
  },
  {
    google_group: {
      group_id: "test_id_3"
    }
    git_hub_team: {
      org_id: 1
      team_id: 4
    }
  }
]
`,
			wantGoogleGroupToGitHubMapper: &GroupMapper{
				mappings: map[string][]string{
					"test_id_1": {"1:2", "1:3"},
					"test_id_2": {"1:4"},
					"test_id_3": {"1:4"},
				},
			},
			wantGitHubToGoogleGroupMapper: &GroupMapper{
				mappings: map[string][]string{
					"1:2": {"test_id_1"},
					"1:3": {"test_id_1"},
					"1:4": {"test_id_2", "test_id_3"},
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
			td := t.TempDir()

			// Create a defaultWritePath in the temp dir.
			tempFile, err := os.CreateTemp(td, defaultWritePath)
			if err != nil {
				t.Fatal("failed to create tempFile: %w", err)
			}
			defer os.Remove(tempFile.Name())

			// Write textproto to temp dir.
			_, err = tempFile.WriteString(tc.content)
			if err != nil {
				t.Fatal("failed to write tempFile: %w", err)
			}

			// if tc.fileReadPath is provided, default file path
			// won't be used, this enable test to read for non-exist
			// path.
			if tc.fileReadpath == "" {
				tc.fileReadpath = tempFile.Name()
			}
			BiDirectionalGroupMapper, err := NewBidirectionaGroupMapper(tc.fileReadpath)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(BiDirectionalGroupMapper.SourceMapper.mappings, tc.wantGoogleGroupToGitHubMapper.mappings, cmp.AllowUnexported()); diff != "" {
				t.Errorf("got unexpected GoogleGroupToGitHubMapper:\n%s", diff)
			}
			if diff := cmp.Diff(BiDirectionalGroupMapper.TargetMapper.mappings, tc.wantGitHubToGoogleGroupMapper.mappings, cmp.AllowUnexported()); diff != "" {
				t.Errorf("got unexpected GitHubToGoogleGroupMapper:\n%s", diff)
			}
		})
	}
}

func TestNewGoogleGroupGitHubUserMapper(t *testing.T) {
	t.Parallel()
	defaultWritePath := "test.textproto"
	cases := []struct {
		name                              string
		fileReadpath                      string
		content                           string
		wantGoogleGroupToGitHubUserMapper *GoogleGroupGitHubUserMapper
		wantErr                           string
	}{
		{
			name: "success",
			content: `
mappings: [
  {
    google_user_email: "src_id_1"
	git_hub_user_id: "dst_id_1"
  },
  {
    google_user_email: "src_id_2"
	git_hub_user_id: "dst_id_2"
  }
]
`,
			wantGoogleGroupToGitHubUserMapper: &GoogleGroupGitHubUserMapper{
				mappings: map[string]string{
					"src_id_1": "dst_id_1",
					"src_id_2": "dst_id_2",
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

			ctx := context.Background()

			td := t.TempDir()

			// Create a defaultWritePath in the temp dir.
			tempFile, err := os.CreateTemp(td, defaultWritePath)
			if err != nil {
				t.Fatal("failed to create tempFile: %w", err)
			}
			defer os.Remove(tempFile.Name())

			// Write textproto to temp dir.
			_, err = tempFile.WriteString(tc.content)
			if err != nil {
				t.Fatal("failed to write tempFile: %w", err)
			}

			// if tc.fileReadPath is provided, default file path
			// won't be used, this enable test to read for non-exist
			// path.
			if tc.fileReadpath == "" {
				tc.fileReadpath = tempFile.Name()
			}
			gotGGToGH, err := NewUserMapper(ctx, tc.fileReadpath)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(gotGGToGH.mappings, tc.wantGoogleGroupToGitHubUserMapper.mappings, cmp.AllowUnexported()); diff != "" {
				t.Errorf("got unexpected GoogleGroupToGitHubMapper:\n%s", diff)
			}
		})
	}
}
