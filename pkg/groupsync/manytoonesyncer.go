// Copyright 2025 The Authors (see AUTHORS file)
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
	"strings"

	"github.com/abcxyz/pkg/logging"
)

// ManyToOneSyncer adheres to the v1alpha3.GroupSyncer interface.
// This syncer allows for syncing many source groups from different source
// systems to one target group in a single target system.
// It adheres to the following policy when syncing a source group ID:
//
//  1. Find the target group that the given source group maps to.
//  2. Find all source groups that map to the target group and forms the union
//     of all descendants from amongst those groups.
//  3. This set of source users is then mapped to their corresponding target users
//     forming the target member set.
//  4. The target member set is then synced to the target group.
type ManyToOneSyncer struct {
	name               string   // A descriptiven name or identifier for the syncer.
	sourceSystems      []string // The key set of sourceGroupReaders.
	targetSystem       string
	sourceGroupReaders map[string]GroupReader // Key represents source system.
	targetGroupWriter  GroupWriter
	sourceGroupMapper  OneToOneGroupMapper
	targetGroupMapper  OneToManyGroupMapper
	userMappers        map[string]UserMapper // Key represents source system.
}

// NewManyToOneSyncer creates a new ManyToOneSyncer.
func NewManyToOneSyncer(
	name string,
	targetSystem string,
	sourceGroupClients map[string]GroupReader,
	targetGroupClient GroupWriter,
	sourceGroupMapper OneToOneGroupMapper,
	targetGroupMapper OneToManyGroupMapper,
	userMappers map[string]UserMapper,
) *ManyToOneSyncer {
	// Abstract the source systems from the sourceGroupClients map.
	sources := make([]string, 0, len(sourceGroupClients))
	for k := range sourceGroupClients {
		sources = append(sources, k)
	}
	return &ManyToOneSyncer{
		name:               name,
		sourceSystems:      sources,
		targetSystem:       targetSystem,
		sourceGroupReaders: sourceGroupClients,
		targetGroupWriter:  targetGroupClient,
		sourceGroupMapper:  sourceGroupMapper,
		targetGroupMapper:  targetGroupMapper,
		userMappers:        userMappers,
	}
}

// SourceSystem returns the name of the source group system.
func (f *ManyToOneSyncer) SourceSystem() string {
	return strings.Join(f.sourceSystems, ",")
}

// TargetSystem returns the name of the target group system.
func (f *ManyToOneSyncer) TargetSystem() string {
	return f.targetSystem
}

// Name returns the syncer name.
func (f *ManyToOneSyncer) Name() string {
	return f.name
}

// Sync syncs the source group with the given ID to the target group system.
func (f *ManyToOneSyncer) Sync(ctx context.Context, sourceGroupID string) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "starting sync", "source_group_id", sourceGroupID)
	// Get target group ID for this source group ID
	targetGroup, err := f.sourceGroupMapper.Mapping(ctx, sourceGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to map source group id to target group id",
			"source_group_id", sourceGroupID,
			"error", err,
		)
		return fmt.Errorf("error fetching target group id: %s, %w", sourceGroupID, err)
	}
	logger.InfoContext(ctx, "found the following target group id to sync",
		"source_group_id", sourceGroupID,
		"target_group", targetGroup,
	)

	return f.sync(ctx, targetGroup.GroupID)
}

func (f *ManyToOneSyncer) sync(ctx context.Context, targetGroupID string) error {
	logger := logging.FromContext(ctx)

	// Get all source group mappings associated with the current target Group ID
	sourceGroupMappings, err := f.targetGroupMapper.Mappings(ctx, targetGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed getting one or more source group mappings for target group id",
			"target_group_id", targetGroupID,
			"source_group_mappings", sourceGroupMappings,
			"error", err,
		)
		return fmt.Errorf("error getting associated source group ids: %w", err)
	}
	logger.InfoContext(ctx, "found source group mappings for target Group id",
		"target_group_id", targetGroupID,
		"source_group_mappings", sourceGroupMappings,
	)

	// Get the union of all users that are members of each source group
	sourceUsers, err := f.sourceUsers(ctx, sourceGroupMappings)
	if err != nil {
		return fmt.Errorf("error getting source users: %w", err)
	}
	logger.InfoContext(ctx, "found descendant(s) for source group id(s)",
		"source_group_mappings", sourceGroupMappings,
		"source_users", sourceUsers,
	)

	// Map each source user to their corresponding target user
	targetUsers, err := f.targetUsers(ctx, sourceUsers)
	if err != nil {
		return fmt.Errorf("error getting one or more target users: %w", err)
	}
	logger.InfoContext(ctx, "mapped source users to target users",
		"source_users", sourceUsers,
		"target_users", targetUsers,
	)

	// map each targetUser to Member type
	targetMembers := make([]Member, 0, len(targetUsers))
	for _, user := range targetUsers {
		targetMembers = append(targetMembers, &UserMember{Usr: user})
	}

	// targetMembers is now the canonical set of members for the target group ID.
	// Set the target group's members to targetMembers.
	logger.InfoContext(ctx, "setting target group id members to target users",
		"target_group_id", targetGroupID,
		"target_users", targetUsers,
	)
	if err := f.targetGroupWriter.SetMembers(ctx, targetGroupID, targetMembers); err != nil {
		logger.ErrorContext(ctx, "failed setting target group members",
			"target_group_id", targetGroupID,
			"error", err,
		)
		return fmt.Errorf("error setting members to target group %s: %w", targetGroupID, err)
	}

	return nil
}

// SyncAll syncs all source groups that this GroupSyncer is aware of to the target system.
func (f *ManyToOneSyncer) SyncAll(ctx context.Context) error {
	targetGroupIDs, err := f.targetGroupMapper.AllGroupIDs(ctx)
	if err != nil {
		return fmt.Errorf("error fetching source group ids: %w", err)
	}
	if err := concurrentSyncFunc(ctx, targetGroupIDs, f.sync); err != nil {
		return fmt.Errorf("failed to sync one or more ids: %w", err)
	}
	return nil
}

// returns an empty list if none were found.
func (f *ManyToOneSyncer) sourceUsers(ctx context.Context, sourceGroupMappings []Mapping) ([]*User, error) {
	userMap := make(map[string]*User)
	for _, sourceGroupMapping := range sourceGroupMappings {
		system := sourceGroupMapping.System
		if system == "" {
			return nil, fmt.Errorf("missing source system for source group reader: %s", sourceGroupMapping)
		}
		groupReader, exist := f.sourceGroupReaders[system]
		if !exist {
			return nil, fmt.Errorf("source group reader not found: %s", sourceGroupMapping)
		}
		sourceUsers, err := groupReader.Descendants(ctx, sourceGroupMapping.GroupID)
		if err != nil {
			return nil, fmt.Errorf("error fetching source group users: %s, %w", sourceGroupMapping, err)
		}
		for _, sourceUser := range sourceUsers {
			mappedUser, exists := userMap[sourceUser.ID]
			metadata := sourceGroupMapping.Metadata
			if exists {
				if metadata == nil {
					metadata = mappedUser.Metadata
				} else {
					metadata = metadata.Combine(mappedUser.Metadata)
				}
			}
			userMap[sourceUser.ID] = &User{
				ID:         sourceUser.ID,
				System:     system,
				Attributes: sourceUser.Attributes,
				Metadata:   metadata,
			}
		}
	}
	users := make([]*User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}
	return users, nil
}

// returns an empty list if none were found.
func (f *ManyToOneSyncer) targetUsers(ctx context.Context, sourceUsers []*User) ([]*User, error) {
	targetUsers := make([]*User, 0, len(sourceUsers))
	for _, sourceUser := range sourceUsers {
		system := sourceUser.System
		if system == "" {
			return nil, fmt.Errorf("missing source system for source user id %s", sourceUser.ID)
		}
		userMapper, exist := f.userMappers[system]
		if !exist {
			return nil, fmt.Errorf("user mapper not found for system %s, source user id %s", sourceUser.System, sourceUser.ID)
		}
		targetUser, err := userMapper.MappedUser(ctx, sourceUser)
		if errors.Is(err, ErrTargetUserIDNotFound) {
			// if there is no mapping for the target user we will just skip them.
			// it happens when the user is removed from the source system.
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error mapping source user id %s to target user id: %w", sourceUser.ID, err)
		}
		targetUsers = append(targetUsers, targetUser)
	}
	return targetUsers, nil
}
