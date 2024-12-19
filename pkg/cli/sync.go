// Copyright 2022 Google LLC
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

package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/team-link/pkg/common"
)

var (
	_ cli.Command = (*SyncCommand)(nil)
)

type SyncCommand struct {
	cli.BaseCommand

	mapping string
	config  string
}

func (c *SyncCommand) Desc() string {
	return `Sync membership information`
}

func (c *SyncCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Sync membership information from source and target system.

  Sync membership from google group to GitHub

  tlctl sync run \
	-mapping mapping.textproto \
	-config config.textproto 
`
}

func (c *SyncCommand) Flags() *cli.FlagSet {
	set := c.NewFlagSet()

	// Command options
	f := set.NewSection("COMMAND OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "mapping",
		Target:  &c.mapping,
		Aliases: []string{"m"},
		Example: "mapping.textproto",
		Usage:   `The textproto file that includes group and user mapping info`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "config",
		Target:  &c.config,
		Aliases: []string{"c"},
		Example: "GitHub",
		Usage:   `The textproto file for teamlink configs.`,
	})

	set.AfterParse(func(merr error) error {
		if c.mapping == "" {
			merr = errors.Join(merr, fmt.Errorf("mapping file is not provided"))
		}
		if c.config == "" {
			merr = errors.Join(merr, fmt.Errorf("config file is not provided"))
		}
		return merr
	})

	return set
}

func (c *SyncCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %q", args)
	}

	if err := common.Sync(ctx, c.mapping, c.config); err != nil {
		return fmt.Errorf("failed to sync membership: %w", err)
	}

	return nil
}
