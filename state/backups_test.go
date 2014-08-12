// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "launchpad.net/gocheck"

	"github.com/juju/juju/state"
	"github.com/juju/juju/state/backups/metadata"
	"github.com/juju/juju/version"
)

type backupSuite struct {
	ConnSuite
}

var _ = gc.Suite(&backupSuite{})

func (s *backupSuite) metadata(c *gc.C) *metadata.Metadata {
	origin := metadata.NewOrigin(
		s.State.EnvironTag().Id(),
		"0",
		"localhost",
		version.Current.Number,
	)
	meta := metadata.NewMetadata(*origin, "", nil)
	meta.Finish(int64(42), "some hash", "", nil)
	return meta
}

func (s *backupSuite) checkMetadata(
	c *gc.C, metadata, expected *metadata.Metadata, id string,
) {
	if id != "" {
		c.Check(metadata.ID(), gc.Equals, id)
	}
	c.Check(metadata.Notes(), gc.Equals, expected.Notes())
	c.Check(metadata.Timestamp().Unix(), gc.DeepEquals, expected.Timestamp().Unix())
	c.Check(metadata.Checksum(), gc.Equals, expected.Checksum())
	c.Check(metadata.ChecksumFormat(), gc.Equals, expected.ChecksumFormat())
	c.Check(metadata.Size(), gc.Equals, expected.Size())
	c.Check(metadata.Origin(), gc.DeepEquals, expected.Origin())
	c.Check(metadata.Stored(), gc.DeepEquals, expected.Stored())
}

//---------------------------
// getBackupMetadata()

func (s *backupSuite) TestBackupsGetBackupMetadataFound(c *gc.C) {
	expected := s.metadata(c)
	id, err := state.AddBackupMetadata(s.State, expected)
	c.Assert(err, gc.IsNil)

	metadata, err := state.GetBackupMetadata(s.State, id)
	c.Check(err, gc.IsNil)

	s.checkMetadata(c, metadata, expected, id)
}

func (s *backupSuite) TestBackupsGetBackupMetadataNotFound(c *gc.C) {
	_, err := state.GetBackupMetadata(s.State, "spam")
	c.Check(err, jc.Satisfies, errors.IsNotFound)
}

//---------------------------
// addBackupMetadata()

func (s *backupSuite) TestBackupsAddBackupMetadataSuccess(c *gc.C) {
	expected := s.metadata(c)
	id, err := state.AddBackupMetadata(s.State, expected)
	c.Check(err, gc.IsNil)

	metadata, err := state.GetBackupMetadata(s.State, id)
	c.Assert(err, gc.IsNil)

	s.checkMetadata(c, metadata, expected, id)
}

func (s *backupSuite) TestBackupsAddBackupMetadataGeneratedID(c *gc.C) {
	expected := s.metadata(c)
	expected.SetID("spam")
	id, err := state.AddBackupMetadata(s.State, expected)
	c.Check(err, gc.IsNil)

	c.Check(id, gc.Not(gc.Equals), "spam")
}

func (s *backupSuite) TestBackupsAddBackupMetadataEmpty(c *gc.C) {
	original := metadata.Metadata{}
	c.Assert(original.Timestamp(), gc.NotNil)
	_, err := state.AddBackupMetadata(s.State, &original)

	c.Check(err, gc.NotNil)
}

func (s *backupSuite) TestBackupsAddBackupMetadataAlreadyExists(c *gc.C) {
	expected := s.metadata(c)
	id, err := state.AddBackupMetadata(s.State, expected)
	c.Assert(err, gc.IsNil)
	err = state.AddBackupMetadataID(s.State, expected, id)

	c.Check(err, jc.Satisfies, errors.IsAlreadyExists)
}

//---------------------------
// setBackupStored()

func (s *backupSuite) TestBackupsSetBackupStoredSuccess(c *gc.C) {
	original := s.metadata(c)
	id, err := state.AddBackupMetadata(s.State, original)
	c.Check(err, gc.IsNil)
	metadata, err := state.GetBackupMetadata(s.State, id)
	c.Assert(err, gc.IsNil)
	c.Assert(metadata.Stored(), gc.Equals, false)

	err = state.SetBackupStored(s.State, id)
	c.Check(err, gc.IsNil)

	metadata, err = state.GetBackupMetadata(s.State, id)
	c.Assert(err, gc.IsNil)
	c.Assert(metadata.Stored(), gc.Equals, true)
}

func (s *backupSuite) TestBackupsSetBackupStoredNotFound(c *gc.C) {
	err := state.SetBackupStored(s.State, "spam")

	c.Check(err, jc.Satisfies, errors.IsNotFound)
}
