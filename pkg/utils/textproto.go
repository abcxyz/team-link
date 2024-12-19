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

package utils

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/prototext"

	api "github.com/abcxyz/team-link/apis/v1alpha3/proto"
	tltypes "github.com/abcxyz/team-link/internal"
)

func ParseMappingTextProto(ctx context.Context, file string) (*api.TeamLinkMappings, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	var tm api.TeamLinkMappings
	if err := prototext.Unmarshal(b, &tm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping file: %w", err)
	}
	return &tm, nil
}

func ParseConfigTextProto(ctx context.Context, file string) (*api.TeamLinkConfig, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}
	var c api.TeamLinkConfig
	if err := prototext.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal teamlink config file: %w", err)
	}
	return &c, nil
}

// GetSrcTargetSystemType parse source and target system typle from teamlink config.
func GetSrcTargetSystemType(tlconfig *api.TeamLinkConfig) (string, string, error) {
	var sourceType string
	switch tlconfig.GetSourceConfig().GetConfig().(type) {
	case *api.SourceConfig_GoogleGroupsConfig:
		sourceType = tltypes.SystemTypeGoogleGroups
	default:
		sourceType = ""
	}

	var targetType string
	switch tlconfig.GetTargetConfig().GetConfig().(type) {
	case *api.TargetConfig_GithubConfig:
		targetType = tltypes.SystemTypeGitHub
	case *api.TargetConfig_GitlabConfig:
		targetType = tltypes.SystemTypeGitLab
	default:
		targetType = ""
	}

	if sourceType == "" || targetType == "" {
		return "", "", fmt.Errorf("source system and target system config not provided")
	}
	return sourceType, targetType, nil
}
