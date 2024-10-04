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
	sourceSystem, targetSystem string,
	sourceGroupClient GroupReader,
	targetGroupClient GroupWriter,
	sourceGroupMapper OneToManyGroupMapper,
	targetGroupMapper OneToManyGroupMapper,
	userMapper UserMapper,
) *ManyToManySyncer {
	return &ManyToManySyncer{
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

// Sync syncs the source group with the given ID to the target group system.
func (f *ManyToManySyncer) Sync(ctx context.Context, sourceGroupID string) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "starting sync", "source_group_id", sourceGroupID)
	// get target group IDs for this source group ID
	targetGroupIDs, err := f.sourceGroupMapper.MappedGroupIDs(ctx, sourceGroupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to map source group ID to target groups IDs",
			"source_group_id", sourceGroupID,
			"error", err,
		)
		return fmt.Errorf("error fetching target group IDs: %s, %w", sourceGroupID, err)
	}
	logger.InfoContext(ctx, "found the following target group IDs to sync",
		"source_group_id", sourceGroupID,
		"target_group_ids", targetGroupIDs,
	)

	var merr error
	for _, targetGroupID := range targetGroupIDs {
		logger.InfoContext(ctx, "syncing target group ID",
			"target_group_id", targetGroupID,
		)
		// get all source group IDs associated with the current target GroupID
		sourceGroupIDs, err := f.targetGroupMapper.MappedGroupIDs(ctx, targetGroupID)
		if err != nil {
			logger.ErrorContext(ctx, "failed getting one ore more source group IDs for target group ID",
				"target_group_id", targetGroupID,
				"source_group_ids", sourceGroupIDs,
				"error", err,
			)
			merr = errors.Join(merr, fmt.Errorf("error getting associated source group ids: %w", err))
			// cannot map this targetGroupID successfully so abort and move on to the next one
			continue
		}
		logger.InfoContext(ctx, "found source group ID(s) for target Group ID",
			"target_group_id", targetGroupID,
			"source_group_ids", sourceGroupIDs,
		)

		// get the union of all users that are members of each source group
		sourceUsers, err := f.sourceUsers(ctx, sourceGroupIDs)
		sourceUserIds := userIDs(sourceUsers)
		if err != nil {
			logger.ErrorContext(ctx, "failed getting one or more source users for source group IDs",
				"source_group_ids", sourceGroupIDs,
				"source_user_ids", sourceUserIds,
				"error", err,
			)
			merr = errors.Join(merr, fmt.Errorf("error getting one or more source users: %w", err))
			// cannot map this targetGroupID successfully so abort and move on to the next one
			continue
		}
		logger.InfoContext(ctx, "found descendant(s) for source group ID(s)",
			"source_group_ids", sourceGroupIDs,
			"source_user_ids", sourceUserIds,
		)

		// map each source user to their corresponding target user
		targetUsers, err := f.targetUsers(ctx, sourceUsers)
		targetUserIds := userIDs(targetUsers)
		if err != nil {
			logger.ErrorContext(ctx, "failed mapping one or more source users to their target user",
				"source_user_ids", sourceUserIds,
				"target_user_ids", targetUserIds,
				"error", err,
			)
			merr = errors.Join(merr, fmt.Errorf("error getting one or more target users: %w", err))
			// cannot map this targetGroupID successfully so abort and move on to the next one
			continue
		}
		logger.InfoContext(ctx, "mapped source users to target users",
			"source_user_ids", sourceUserIds,
			"target_user_ids", targetUserIds,
		)

		// map each targetUser to Member type
		targetMembers := make([]Member, 0, len(targetUsers))
		for _, user := range targetUsers {
			targetMembers = append(targetMembers, &UserMember{Usr: user})
		}

		// targetMembers is now the canonical set of members for the target group ID.
		// Set the target group's members to targetMembers.
		logger.InfoContext(ctx, "setting target group ID members to target users",
			"target_group_id", targetGroupID,
			"target_user_ids", targetUserIds,
		)
		if err := f.targetGroupReadWriter.SetMembers(ctx, targetGroupID, targetMembers); err != nil {
			logger.ErrorContext(ctx, "failed setting target group members",
				"target_group_id", targetGroupID,
				"error", err,
			)
			merr = fmt.Errorf("error setting members to target group %s: %w", targetGroupID, err)
		}
	}

	return merr
}

// SyncAll syncs all source groups that this GroupSyncer is aware of to the target system.
func (f *ManyToManySyncer) SyncAll(ctx context.Context) error {
	sourceGroupIDs, err := f.sourceGroupMapper.AllGroupIDs(ctx)
	if err != nil {
		return fmt.Errorf("error fetching source group IDs: %w", err)
	}
	if err := ConcurrentSync(ctx, f, sourceGroupIDs); err != nil {
		return fmt.Errorf("failed to sync one or more IDs: %w", err)
	}
	return nil
}

func (f *ManyToManySyncer) sourceUsers(ctx context.Context, sourceGroupIDs []string) ([]*User, error) {
	var merr error
	userMap := make(map[string]*User)
	for _, sourceGroupID := range sourceGroupIDs {
		sourceUsers, err := f.sourceGroupReader.Descendants(ctx, sourceGroupID)
		if err != nil {
			merr = errors.Join(merr, fmt.Errorf("error fetching source group users: %s, %w", sourceGroupID, err))
			continue
		}
		for _, sourceUser := range sourceUsers {
			userMap[sourceUser.ID] = sourceUser
		}
	}
	users := make([]*User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}
	return users, merr
}

func (f *ManyToManySyncer) targetUsers(ctx context.Context, sourceUsers []*User) ([]*User, error) {
	var merr error
	targetUsers := make([]*User, 0, len(sourceUsers))
	for _, sourceUser := range sourceUsers {
		targetUserID, err := f.userMapper.MappedUserID(ctx, sourceUser.ID)
		if err != nil {
			merr = fmt.Errorf("error mapping source user id %s to target user id: %w", sourceUser.ID, err)
			continue
		}
		targetUsers = append(targetUsers, &User{ID: targetUserID})
	}
	return targetUsers, merr
}

func userIDs(users []*User) []string {
	ids := make([]string, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	return ids
}
