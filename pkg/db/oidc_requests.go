/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/trustbloc/edge-adapter/pkg/presentationex"
)

//nolint:lll
const (
	sqlInsertOIDCRequest            = "insert into oidc_request (end_user_id, relying_party_id, scopes, scopes_hash, pres_def) values (?, ?, ?, ?, ?)"
	sqlUpdateOIDCRequest            = `update oidc_request set end_user_id = ?, relying_party_id = ?, scopes = ?, scopes_hash = ?, pres_def = ? where id = ?`
	sqlSelectOIDCRequestByEndUserID = `
select oidc_request.*
from oidc_request
inner join end_user
	on oidc_request.end_user_id = end_user.id
inner join relying_party
	on oidc_request.relying_party_id = relying_party.id
where end_user.sub = ?
and relying_party.client_id = ?
and oidc_request.scopes_hash = ?
`
)

// OIDCRequest is a Relying Party's OIDC request for user data.
type OIDCRequest struct {
	ID             int64
	EndUserID      int64
	RelyingPartyID int64
	Scopes         []string
	PresDef        *presentationex.PresentationDefinitions
}

// NewOIDCRequests returns a new OIDCRequests.
func NewOIDCRequests(db *sql.DB) *OIDCRequests {
	return &OIDCRequests{DB: db}
}

// OIDCRequests is an OIDCRequest DAO.
type OIDCRequests struct {
	DB *sql.DB
}

// Insert this oidc request to the DB.
func (o *OIDCRequests) Insert(r *OIDCRequest) error {
	presDef, err := nullable(r.PresDef)
	if err != nil {
		return err
	}

	scopesHash := base64.StdEncoding.
		WithPadding(base64.NoPadding).
		EncodeToString(sha256.New().Sum([]byte(strings.Join(r.Scopes, ""))))

	result, err := o.DB.Exec(
		sqlInsertOIDCRequest, r.EndUserID, r.RelyingPartyID, strings.Join(r.Scopes, ","), scopesHash, presDef)
	if err != nil {
		return fmt.Errorf("failed to insert oidc request %+v : %w", r, err)
	}

	r.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to retrieve last insert id : %w", err)
	}

	return nil
}

// FindBySubRPClientIDAndScopes fetches the OIDC request sent by `clientID` with `sub` as the user subject.
// TODO FindBySubRPClientIDAndScopes should return a list of oidc requests with just sub and clientID as args.
//  How can we triangulate a single record? https://github.com/trustbloc/edge-adapter/issues/30
func (o *OIDCRequests) FindBySubRPClientIDAndScopes(sub, clientID string, scopes []string) (*OIDCRequest, error) {
	var (
		scopesInDB string
		presDef    string
		scopesHash = base64.StdEncoding.
				WithPadding(base64.NoPadding).
				EncodeToString(sha256.New().Sum([]byte(strings.Join(scopes, ""))))
	)

	result := &OIDCRequest{}

	err := o.DB.QueryRow(sqlSelectOIDCRequestByEndUserID, sub, clientID, scopesHash).
		Scan(&result.ID, &result.EndUserID, &result.RelyingPartyID, &scopesInDB, &scopesHash, &presDef)
	if err != nil {
		return nil, fmt.Errorf("failed to query oidc request by user sub : %w", err)
	}

	result.Scopes = strings.Split(scopesInDB, ",")
	result.PresDef = &presentationex.PresentationDefinitions{}

	err = json.Unmarshal([]byte(presDef), &result.PresDef)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal presentation definitions : %w", err)
	}

	return result, nil
}

// Update updates the oidc request's fields in the datasource.
func (o *OIDCRequests) Update(r *OIDCRequest) error {
	presDef, err := nullable(r.PresDef)
	if err != nil {
		return err
	}

	scopesHash := base64.StdEncoding.
		WithPadding(base64.NoPadding).
		EncodeToString(sha256.New().Sum([]byte(strings.Join(r.Scopes, ""))))

	result, err := o.DB.Exec(
		sqlUpdateOIDCRequest, r.EndUserID, r.RelyingPartyID, strings.Join(r.Scopes, ","), scopesHash, presDef, r.ID)
	if err != nil {
		return fmt.Errorf("update oidc_request : %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update oidc_request : failed to fetch num affected rows : %w", err)
	}

	if n == 0 {
		return fmt.Errorf("update oidc_request : no such oidc request with id=%d", r.ID)
	}

	return nil
}

func nullable(pd *presentationex.PresentationDefinitions) (*sql.NullString, error) {
	if pd == nil {
		return &sql.NullString{}, nil
	}

	bits, err := json.Marshal(pd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal presentation definition : %w", err)
	}

	return &sql.NullString{
		String: string(bits),
		Valid:  true,
	}, nil
}