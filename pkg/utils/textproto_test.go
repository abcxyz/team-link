package utils

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/abcxyz/pkg/testutil"
	api "github.com/abcxyz/team-link/apis/v1alpha3/proto"
)

func TestParseMappingTextProto(t *testing.T) {
	t.Parallel()
	defaultWritePath := "test.textproto"
	cases := []struct {
		name                 string
		fileReadpath         string
		content              string
		wantTeamLinkMappings *api.TeamLinkMappings
		wantErr              string
	}{
		{
			name: "success",
			content: `
group_mappings {
  mappings: [
    {
      google_groups: {
	    group_id: "test_id_1"
	  }
	  github: {
	    org_id: 1
		team_id: 2
	  }
	},
    {
      google_groups: {
	    group_id: "test_id_2"
	  }
	  github: {
	    org_id: 1
		team_id: 3
	  }
	}
  ]
}
user_mappings {
  mappings: [
    {
      source: "foo@example.com"
	  target: "user_1"
	},
	{
	  source: "bar@example.com"
	  target: "user_2"
	}
  ]
}
`,
			wantTeamLinkMappings: &api.TeamLinkMappings{
				GroupMappings: &api.GroupMappings{
					Mappings: []*api.GroupMapping{
						{
							Source: &api.GroupMapping_GoogleGroups{
								GoogleGroups: &api.GoogleGroups{
									GroupId: "test_id_1",
								},
							},
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:  1,
									TeamId: 2,
								},
							},
						},
						{
							Source: &api.GroupMapping_GoogleGroups{
								GoogleGroups: &api.GoogleGroups{
									GroupId: "test_id_2",
								},
							},
							Target: &api.GroupMapping_Github{
								Github: &api.GitHub{
									OrgId:  1,
									TeamId: 3,
								},
							},
						},
					},
				},
				UserMappings: &api.UserMappings{
					Mappings: []*api.UserMapping{
						{
							Source: "foo@example.com",
							Target: "user_1",
						},
						{
							Source: "bar@example.com",
							Target: "user_2",
						},
					},
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
			res, err := ParseMappingTextProto(ctx, tc.fileReadpath)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(res.GetGroupMappings().GetMappings(), tc.wantTeamLinkMappings.GetGroupMappings().GetMappings(), cmpopts.IgnoreUnexported(api.GroupMapping{}, api.GoogleGroups{}, api.GitHub{})); diff != "" {
				t.Errorf("got unexpected GroupMappings:\n%s", diff)
			}
			if diff := cmp.Diff(res.GetUserMappings().GetMappings(), tc.wantTeamLinkMappings.GetUserMappings().GetMappings(), cmpopts.IgnoreUnexported(api.UserMapping{})); diff != "" {
				t.Errorf("got unexpected UserMappings:\n%s", diff)
			}
		})
	}
}

func TestParseConfigTextProto(t *testing.T) {
	t.Parallel()
	defaultWritePath := "test.textproto"
	cases := []struct {
		name               string
		fileReadpath       string
		content            string
		wantTeamLinkConfig *api.TeamLinkConfig
		wantErr            string
	}{
		{
			name: "success",
			content: `
source_config {
    google_groups_config {}
}
target_config {
    github_config {
        enterprise_url: "https://github.com",
        static_auth {
            from_environment: "TEAM_LINK_GITHUB_TOKEN"
        }
    }
}
`,
			wantTeamLinkConfig: &api.TeamLinkConfig{
				SourceConfig: &api.SourceConfig{
					Config: &api.SourceConfig_GoogleGroupsConfig{
						GoogleGroupsConfig: &api.GoogleGroupsConfig{},
					},
				},
				TargetConfig: &api.TargetConfig{
					Config: &api.TargetConfig_GithubConfig{
						GithubConfig: &api.GitHubConfig{
							EnterpriseUrl: "https://github.com",
							Authentication: &api.GitHubConfig_StaticAuth{
								StaticAuth: &api.StaticToken{
									FromEnvironment: "TEAM_LINK_GITHUB_TOKEN",
								},
							},
						},
					},
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
			wantErr: "failed to unmarshal teamlink config file",
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
			res, err := ParseConfigTextProto(ctx, tc.fileReadpath)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(res.GetSourceConfig(), tc.wantTeamLinkConfig.GetSourceConfig(), cmpopts.IgnoreUnexported(api.SourceConfig{}, api.GoogleGroupsConfig{})); diff != "" {
				t.Errorf("got unexpected SourceConfig:\n%s", diff)
			}
			if diff := cmp.Diff(res.GetTargetConfig(), tc.wantTeamLinkConfig.GetTargetConfig(), cmpopts.IgnoreUnexported(api.TargetConfig{}, api.GitHubConfig{}, api.StaticToken{})); diff != "" {
				t.Errorf("got unexpected TargetConfig:\n%s", diff)
			}
		})
	}
}
