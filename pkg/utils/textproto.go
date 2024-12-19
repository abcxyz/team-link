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
	switch tlconfig.SourceConfig.Config.(type) {
	case *api.SourceConfig_GoogleGroupsConfig:
		sourceType = tltypes.SystemTypeGoogleGroups
	default:
		sourceType = ""
	}

	var targetType string
	switch tlconfig.TargetConfig.Config.(type) {
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
