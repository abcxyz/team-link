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
	"context"
	"fmt"
	"sort"
	"sync"
)

type testReadWriteGroupClient struct {
	groups          map[string]*Group
	groupMembers    map[string][]Member
	users           map[string]*User
	descendantsErrs map[string]error
	getGroupErrs    map[string]error
	getMembersErrs  map[string]error
	getUserErrs     map[string]error
	setMembersErrs  map[string]error
	mutex           sync.RWMutex
}

func (tc *testReadWriteGroupClient) Descendants(ctx context.Context, groupID string) ([]*User, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.descendantsErrs[groupID]; ok {
		return nil, err
	}
	return Descendants(ctx, groupID, tc.GetMembers)
}

func (tc *testReadWriteGroupClient) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.getGroupErrs[groupID]; ok {
		return nil, err
	}
	group, ok := tc.groups[groupID]
	if !ok {
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	return group, nil
}

func (tc *testReadWriteGroupClient) GetMembers(ctx context.Context, groupID string) ([]Member, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.getMembersErrs[groupID]; ok {
		return nil, err
	}
	members, ok := tc.groupMembers[groupID]
	if !ok {
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	return members, nil
}

func (tc *testReadWriteGroupClient) GetUser(ctx context.Context, userID string) (*User, error) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	if err, ok := tc.getUserErrs[userID]; ok {
		return nil, err
	}
	user, ok := tc.users[userID]
	if !ok {
		return nil, fmt.Errorf("user %s not found", userID)
	}
	return user, nil
}

func (tc *testReadWriteGroupClient) SetMembers(ctx context.Context, groupID string, members []Member) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	if err, ok := tc.setMembersErrs[groupID]; ok {
		return err
	}
	_, ok := tc.groupMembers[groupID]
	if !ok {
		return fmt.Errorf("group %s not found", groupID)
	}
	// sort members so we have deterministic ordering for comparisons
	sort.Slice(members, func(i, j int) bool {
		u1, err := members[i].User()
		if err != nil {
			// test is broken if this happens
			panic(err)
		}
		u2, err := members[j].User()
		if err != nil {
			// test is broken if this happens
			panic(err)
		}
		return u1.ID < u2.ID
	})
	tc.groupMembers[groupID] = members
	return nil
}

type testOneToManyGroupMapper struct {
	m                   map[string][]Mapping
	allGroupIDsErr      error
	containsGroupIDErrs map[string]error
	mappedGroupIDsErr   map[string]error
}

func (tgm *testOneToManyGroupMapper) AllGroupIDs(ctx context.Context) ([]string, error) {
	if tgm.allGroupIDsErr != nil {
		return nil, tgm.allGroupIDsErr
	}
	ids := make([]string, 0, len(tgm.m))
	for id := range tgm.m {
		ids = append(ids, id)
	}
	return ids, nil
}

func (tgm *testOneToManyGroupMapper) ContainsGroupID(ctx context.Context, groupID string) (bool, error) {
	if err, ok := tgm.containsGroupIDErrs[groupID]; ok {
		return false, err
	}
	_, ok := tgm.m[groupID]
	return ok, nil
}

func (tgm *testOneToManyGroupMapper) MappedGroupIDs(ctx context.Context, groupID string) ([]string, error) {
	if err, ok := tgm.mappedGroupIDsErr[groupID]; ok {
		return nil, err
	}
	mappings, ok := tgm.m[groupID]
	if !ok {
		return nil, fmt.Errorf("group %s not mapped", groupID)
	}
	ids := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		ids = append(ids, mapping.GroupID)
	}
	return ids, nil
}

func (tgm *testOneToManyGroupMapper) Mappings(ctx context.Context, groupID string) ([]Mapping, error) {
	if err, ok := tgm.mappedGroupIDsErr[groupID]; ok {
		return nil, err
	}
	mapping, exist := tgm.m[groupID]
	if !exist {
		return nil, fmt.Errorf("group %s not mapped", groupID)
	}
	return mapping, nil
}

type testUserMapper struct {
	m                map[string]string
	mappedUserIDErrs map[string]error
}

func (tum *testUserMapper) MappedUserID(ctx context.Context, userID string) (string, error) {
	if err, ok := tum.mappedUserIDErrs[userID]; ok {
		return "", err
	}
	id, ok := tum.m[userID]
	if !ok {
		return "", fmt.Errorf("user %s not mapped", userID)
	}
	return id, nil
}

func (tum *testUserMapper) MappedUser(ctx context.Context, user *User) (*User, error) {
	if err, ok := tum.mappedUserIDErrs[user.ID]; ok {
		return nil, err
	}
	u, ok := tum.m[user.ID]
	if !ok {
		return nil, fmt.Errorf("user %s not mapped", user.ID)
	}
	return &User{ID: u}, nil
}

// Implements OneToOneGroupMapper interface.
type testOneToOneGroupMapper struct {
	m                   map[string]Mapping
	allGroupIDsErr      error
	containsGroupIDErrs map[string]error
	mappedGroupIDErr    map[string]error
}

func (tgm *testOneToOneGroupMapper) AllGroupIDs(ctx context.Context) ([]string, error) {
	if tgm.allGroupIDsErr != nil {
		return nil, tgm.allGroupIDsErr
	}
	ids := make([]string, 0, len(tgm.m))
	for id := range tgm.m {
		ids = append(ids, id)
	}
	return ids, nil
}

func (tgm *testOneToOneGroupMapper) ContainsGroupID(ctx context.Context, groupID string) (bool, error) {
	if err, ok := tgm.containsGroupIDErrs[groupID]; ok {
		return false, err
	}
	_, ok := tgm.m[groupID]
	return ok, nil
}

func (tgm *testOneToOneGroupMapper) MappedGroupID(ctx context.Context, groupID string) (string, error) {
	if err, ok := tgm.mappedGroupIDErr[groupID]; ok {
		return "", err
	}
	mapping, ok := tgm.m[groupID]
	if !ok {
		return "", fmt.Errorf("group %s not mapped", groupID)
	}
	return mapping.GroupID, nil
}

func (tgm *testOneToOneGroupMapper) Mapping(ctx context.Context, groupID string) (Mapping, error) {
	if err, ok := tgm.mappedGroupIDErr[groupID]; ok {
		return Mapping{}, err
	}
	mapping, ok := tgm.m[groupID]
	if !ok {
		return Mapping{}, fmt.Errorf("group %s not mapped", groupID)
	}
	return mapping, nil
}
