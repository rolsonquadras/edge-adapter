/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issuer

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hyperledger/aries-framework-go/pkg/doc/util"
	"github.com/hyperledger/aries-framework-go/pkg/doc/verifiable"

	"github.com/trustbloc/edge-adapter/pkg/internal/common/adapterutil"
)

const (
	// DIDConnectCredentialType vc type.
	DIDConnectCredentialType = "DIDConnection"

	// DIDCommInitCredentialType vc type.
	DIDCommInitCredentialType = "DIDCommInit"
)

// ParseWalletResponse parses VP received from the wallet and returns the DIDConnect response.
func ParseWalletResponse(vpBytes []byte) (*DIDConnectCredentialSubject, error) {
	// TODO https://github.com/trustbloc/edge-adapter/issues/87 validate the signature
	pres, err := verifiable.ParsePresentation(vpBytes, verifiable.WithDisabledPresentationProofCheck())
	if err != nil {
		return nil, fmt.Errorf("invalid presentation: %s", err.Error())
	}

	rawCredentials, err := pres.MarshalledCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to parse the credential: %s", err.Error())
	}

	if len(rawCredentials) != 1 {
		return nil, errors.New("there should be only one credential")
	}

	// TODO https://github.com/trustbloc/edge-adapter/issues/87 validate the signature
	cred, err := verifiable.ParseCredential(rawCredentials[0], verifiable.WithDisabledProofCheck())
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential : %s", err.Error())
	}

	if !adapterutil.StringsContains(DIDConnectCredentialType, cred.Types) {
		return nil, fmt.Errorf("vc doesn't contain %s type", DIDConnectCredentialType)
	}

	didConnectVC := &DIDConnectCredential{}

	err = adapterutil.DecodeJSONMarshaller(cred, didConnectVC)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credential : %s", err.Error())
	}

	return didConnectVC.Subject, nil
}

// CreateDIDCommInitCredential creates DIDComm init credential.
func CreateDIDCommInitCredential(docJSON []byte) *verifiable.Credential {
	issued := time.Now()

	// TODO define context
	vc := &verifiable.Credential{
		Context: []string{
			"https://www.w3.org/2018/credentials/v1",
		},
		ID: uuid.New().URN(),
		Types: []string{
			"VerifiableCredential",
			DIDCommInitCredentialType,
		},
		Subject: &DIDCommInitCredentialSubject{
			ID:     uuid.New().String(),
			DIDDoc: docJSON,
		},
		Issuer: verifiable.Issuer{
			ID: uuid.New().URN(),
		},
		Issued: util.NewTime(issued),
	}

	return vc
}