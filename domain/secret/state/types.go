// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"time"

	"github.com/juju/errors"

	coresecrets "github.com/juju/juju/core/secrets"
	domainsecret "github.com/juju/juju/domain/secret"
)

// These structs represent the persistent secretMetadata entity schema in the database.

type secretID struct {
	ID string `db:"id"`
}

type revisionUUID struct {
	UUID string `db:"uuid"`
}

type entityRef struct {
	UUID string `db:"uuid"`
	ID   string `db:"id"`
}

type unit struct {
	UUID     string `db:"uuid"`
	UnitName string `db:"unit_id"`
}

type secretRef struct {
	ID         string `db:"secret_id"`
	SourceUUID string `db:"source_uuid"`
	Revision   int    `db:"revision"`
}

type secretMetadata struct {
	ID             string    `db:"secret_id"`
	Version        int       `db:"version"`
	Description    string    `db:"description"`
	AutoPrune      bool      `db:"auto_prune"`
	RotatePolicyID int       `db:"rotate_policy_id"`
	CreateTime     time.Time `db:"create_time"`
	UpdateTime     time.Time `db:"update_time"`
}

// secretInfo is used because sqlair doesn't seem to like struct embedding.
type secretInfo struct {
	ID           string    `db:"secret_id"`
	Version      int       `db:"version"`
	Description  string    `db:"description"`
	RotatePolicy string    `db:"policy"`
	AutoPrune    bool      `db:"auto_prune"`
	CreateTime   time.Time `db:"create_time"`
	UpdateTime   time.Time `db:"update_time"`

	NextRotateTime   time.Time `db:"next_rotation_time"`
	LatestExpireTime time.Time `db:"latest_expire_time"`
	LatestRevision   int       `db:"latest_revision"`
}

type secretModelOwner struct {
	SecretID string `db:"secret_id"`
	Label    string `db:"label"`
}

type secretApplicationOwner struct {
	SecretID        string `db:"secret_id"`
	ApplicationUUID string `db:"application_uuid"`
	Label           string `db:"label"`
}

type secretUnitOwner struct {
	SecretID string `db:"secret_id"`
	UnitUUID string `db:"unit_uuid"`
	Label    string `db:"label"`
}

type secretOwner struct {
	SecretID  string `db:"secret_id"`
	OwnerID   string `db:"owner_id"`
	OwnerKind string `db:"owner_kind"`
	Label     string `db:"label"`
}

type secretRotate struct {
	SecretID       string    `db:"secret_id"`
	NextRotateTime time.Time `db:"next_rotation_time"`
}

type secretRevision struct {
	ID            string    `db:"uuid"`
	SecretID      string    `db:"secret_id"`
	Revision      int       `db:"revision"`
	Obsolete      bool      `db:"obsolete"`
	PendingDelete bool      `db:"pending_delete"`
	CreateTime    time.Time `db:"create_time"`
	UpdateTime    time.Time `db:"update_time"`
}

type secretRevisionExpire struct {
	RevisionUUID string    `db:"revision_uuid"`
	ExpireTime   time.Time `db:"expire_time"`
}

type secretContent struct {
	RevisionUUID string `db:"revision_uuid"`
	Name         string `db:"name"`
	Content      string `db:"content"`
}

type secretValueRef struct {
	RevisionUUID string `db:"revision_uuid"`
	BackendUUID  string `db:"backend_uuid"`
	RevisionID   string `db:"revision_id"`
}

type secretUnitConsumer struct {
	UnitUUID        string `db:"unit_uuid"`
	SecretID        string `db:"secret_id"`
	Label           string `db:"label"`
	CurrentRevision int    `db:"current_revision"`
}

type secretRemoteUnitConsumer struct {
	UnitID          string `db:"unit_id"`
	SecretID        string `db:"secret_id"`
	CurrentRevision int    `db:"current_revision"`
}

type remoteSecret struct {
	SecretID       string `db:"secret_id"`
	LatestRevision int    `db:"latest_revision"`
}

type secretPermission struct {
	SecretID      string                        `db:"secret_id"`
	RoleID        domainsecret.Role             `db:"role_id"`
	SubjectUUID   string                        `db:"subject_uuid"`
	SubjectTypeID domainsecret.GrantSubjectType `db:"subject_type_id"`
	ScopeUUID     string                        `db:"scope_uuid"`
	ScopeTypeID   domainsecret.GrantScopeType   `db:"scope_type_id"`
}

type secretAccessor struct {
	SecretID      string                        `db:"secret_id"`
	SubjectID     string                        `db:"subject_id"`
	SubjectTypeID domainsecret.GrantSubjectType `db:"subject_type_id"`
	RoleID        domainsecret.Role             `db:"role_id"`
}

type secretAccessScope struct {
	ScopeID     string                      `db:"scope_id"`
	ScopeTypeID domainsecret.GrantScopeType `db:"scope_type_id"`
}

type secrets []secretInfo

func (rows secrets) toSecretMetadata(secretOwners []secretOwner) ([]*coresecrets.SecretMetadata, error) {
	if len(rows) != len(secretOwners) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing secret results")
	}

	result := make([]*coresecrets.SecretMetadata, len(rows))
	for i, row := range rows {
		uri, err := coresecrets.ParseURI(row.ID)
		if err != nil {
			return nil, errors.NotValidf("secret URI %q", row.ID)
		}
		result[i] = &coresecrets.SecretMetadata{
			URI:         uri,
			Version:     row.Version,
			Description: row.Description,
			Label:       secretOwners[i].Label,
			Owner: coresecrets.Owner{
				Kind: coresecrets.OwnerKind(secretOwners[i].OwnerKind),
				ID:   secretOwners[i].OwnerID,
			},
			CreateTime:     row.CreateTime,
			UpdateTime:     row.UpdateTime,
			LatestRevision: row.LatestRevision,
			AutoPrune:      row.AutoPrune,
			RotatePolicy:   coresecrets.RotatePolicy(row.RotatePolicy),
		}
		if tm := row.NextRotateTime; !tm.IsZero() {
			result[i].NextRotateTime = &tm
		}
		if tm := row.LatestExpireTime; !tm.IsZero() {
			result[i].LatestExpireTime = &tm
		}
	}
	return result, nil
}

type secretRevisions []secretRevision
type secretRevisionsExpire []secretRevisionExpire

func (rows secretRevisions) toSecretRevisions(revExpire secretRevisionsExpire) ([]*coresecrets.SecretRevisionMetadata, error) {
	if len(rows) != len(revExpire) {
		// Should never happen.
		return nil, errors.New("row length mismatch composing secret revision results")
	}

	result := make([]*coresecrets.SecretRevisionMetadata, len(rows))
	for i, row := range rows {
		result[i] = &coresecrets.SecretRevisionMetadata{
			Revision:    row.Revision,
			ValueRef:    nil,
			CreateTime:  row.CreateTime,
			UpdateTime:  row.UpdateTime,
			BackendName: nil,
		}
		if tm := revExpire[i].ExpireTime; !tm.IsZero() {
			result[i].ExpireTime = &tm
		}
	}
	return result, nil
}

type secretValues []secretContent

func (rows secretValues) toSecretData() (coresecrets.SecretData, error) {
	result := make(coresecrets.SecretData)
	for _, row := range rows {
		result[row.Name] = row.Content
	}
	return result, nil
}

type secretRemoteUnitConsumers []secretRemoteUnitConsumer

func (rows secretRemoteUnitConsumers) toSecretConsumers() ([]*coresecrets.SecretConsumerMetadata, error) {
	result := make([]*coresecrets.SecretConsumerMetadata, len(rows))
	for i, row := range rows {
		result[i] = &coresecrets.SecretConsumerMetadata{
			CurrentRevision: row.CurrentRevision,
		}
	}
	return result, nil
}

type secretUnitConsumers []secretUnitConsumer

func (rows secretUnitConsumers) toSecretConsumers() ([]*coresecrets.SecretConsumerMetadata, error) {
	result := make([]*coresecrets.SecretConsumerMetadata, len(rows))
	for i, row := range rows {
		result[i] = &coresecrets.SecretConsumerMetadata{
			Label:           row.Label,
			CurrentRevision: row.CurrentRevision,
		}
	}
	return result, nil
}
