// Copyright 2024 Google LLC
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

	"github.com/abcxyz/pkg/logging"
)

// ManyToManySyncer adheres to the v1alpha3.GroupSyncer interface.
// This syncer allows for syncing many source groups to many target groups.
// It adheres to the following policy when syncing a source group ID:
//
//  1. Find all the target groups that the given source group maps to.
//  2. For each of those target groups, it finds all source groups that map to
//     it and forms the union of all descendants from amongst those groups.
//  3. This set of source users is then mapped to their corresponding target users
//     forming the target member set.
//  4. The target member set is then synced to the target group.
type ManyToManySyncer struct {
	name                  string // A descriptiven name or identifier for the syncer.
	sourceSystem          string
	targetSystem          string
	sourceGroupReader     GroupReader
	targetGroupReadWriter GroupWriter
	sourceGroupMapper     OneToManyGroupMapper
	targetGroupMapper     OneToManyGroupMapper
	userMapper            UserMapper
}

// NewManyToManySyncer creates a new ManyToManySyncer.
func NewManyToManySyncer(
	name, sourceSystem, targetSystem string,
	sourceGroupClient GroupReader,
	targetGroupClient GroupWriter,
	sourceGroupMapper OneToManyGroupMapper,
	targetGroupMapper OneToManyGroupMapper,
	userMapper UserMapper,
) *ManyToManySyncer {
	return &ManyToManySyncer{
		name:                  name,
		sourceSystem:          sourceSystem,
		targetSystem:          targetSystem,
		sourceGroupReader:     sourceGroupClient,
		targetGroupReadWriter: targetGroupClient,
		sourceGroupMapper:     sourceGroupMapper,
		targetGroupMapper:     targetGroupMapper,
		userMapper:            userMapper,
	}
}

// SourceSystem returns the name of the source group system.
func (f *ManyToManySyncer) SourceSystem() string {
	return f.sourceSystem
}

// TargetSystem returns the name of the target group system.
func (f *ManyToManySyncer) TargetSystem() string {
	return f.targetSystem
}

// Name returns the syncer name.
func (f *ManyToManySyncer) Name() string {
	return f.name
}

// Sync syncs the source group with the given ID to the target group system.
func (f *ManyToManySyncer) Sync(ctx context.Context, sourceGroupID string) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "starting sync", "source_group_id", sourceGroupID)
	// get target group IDs for this source group ID
	targetGroups, err := f.sourceGroupMapper.Mappings(ctx, sourceGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to map source group ID to target groups IDs",
			"source_group_id", sourceGroupID,
			"error", err,
		)
		return fmt.Errorf("error fetching target group IDs: %s, %w", sourceGroupID, err)
	}
	logger.InfoContext(ctx, "found the following target group IDs to sync",
		"source_group_id", sourceGroupID,
		"target_groups", targetGroups,
	)

	var merr error
	for _, targetGroup := range targetGroups {
		if err := f.syncTargetGroup(ctx, targetGroup.GroupID); err != nil {
			merr = errors.Join(merr, err)
		}
	}

	return merr
}

func (f *ManyToManySyncer) syncTargetGroup(ctx context.Context, targetGroupID string) error {
	logger := logging.FromContext(ctx).With("target_group", targetGroupID)
	ctx = logging.WithLogger(ctx, logger)

	logger.InfoContext(ctx, "starting sync target group")

	// get all source group mappings associated with the current target GroupID
	sourceGroupMappings, err := f.targetGroupMapper.Mappings(ctx, targetGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed getting source group mappings for target group",
			"source_group_mappings", sourceGroupMappings,
			"error", err,
		)
		return fmt.Errorf("error getting associated source groups for target group %s: %w", targetGroupID, err)
	}
	logger.InfoContext(ctx, "found source group mappings",
		"source_group_mappings", sourceGroupMappings,
	)

	// get the union of all users that are members of each source group
	sourceUsers, err := f.sourceUsers(ctx, sourceGroupMappings)
	if err != nil {
		logger.ErrorContext(ctx, "failed getting source users for source groups",
			"source_group_mappings", sourceGroupMappings,
			"error", err,
		)
		return fmt.Errorf("error getting source users for target group %s: %w", targetGroupID, err)
	}
	logger.InfoContext(ctx, "found descendant(s) for source groups",
		"source_group_mappings", sourceGroupMappings,
		"source_users", sourceUsers,
	)

	// map each source user to their corresponding target user
	targetUsers, err := f.targetUsers(ctx, sourceUsers)
	if err != nil {
		logger.ErrorContext(ctx, "failed mapping source user to the target user",
			"error", err,
		)
		return fmt.Errorf("error getting target users for group %s: %w", targetGroupID, err)
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
	logger.InfoContext(ctx, "setting target group ID members to target users",
		"target_users", targetUsers,
	)
	if err := f.targetGroupReadWriter.SetMembers(ctx, targetGroupID, targetMembers); err != nil {
		logger.ErrorContext(ctx, "failed setting target group members",
			"error", err,
		)
		return fmt.Errorf("error setting members to target group %s: %w", targetGroupID, err)
	}
	return nil
}

// SyncAll syncs all source groups that this GroupSyncer is aware of to the target system.
func (f *ManyToManySyncer) SyncAll(ctx context.Context) error {
	targetGroupIDs, err := f.targetGroupMapper.AllGroupIDs(ctx)
	if err != nil {
		return fmt.Errorf("error fetching target group IDs: %w", err)
	}
	if err := concurrentSyncFunc(ctx, targetGroupIDs, f.syncTargetGroup); err != nil {
		return fmt.Errorf("failed to sync one or more IDs: %w", err)
	}
	return nil
}

func (f *ManyToManySyncer) sourceUsers(ctx context.Context, sourceGroupMappings []Mapping) ([]*User, error) {
	userMap := make(map[string]*User)
	for _, sourceGroupMapping := range sourceGroupMappings {
		sourceUsers, err := f.sourceGroupReader.Descendants(ctx, sourceGroupMapping.GroupID)
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

func (f *ManyToManySyncer) targetUsers(ctx context.Context, sourceUsers []*User) ([]*User, error) {
	targetUsers := make([]*User, 0, len(sourceUsers))
	for _, sourceUser := range sourceUsers {
		targetUser, err := f.userMapper.MappedUser(ctx, sourceUser)
		if errors.Is(err, ErrTargetUserIDNotFound) {
			// if there is no mapping for the target user we will just skip them.
			// it happens when the user is removed from the source system or not
			// mapped to target system.
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error mapping source user id %s to target user id: %w", sourceUser.ID, err)
		}
		targetUsers = append(targetUsers, targetUser)
	}
	return targetUsers, nil
}
