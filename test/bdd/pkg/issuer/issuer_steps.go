/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issuer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/cucumber/godog"
	"github.com/google/uuid"
	docdid "github.com/hyperledger/aries-framework-go/pkg/doc/did"
	log "github.com/sirupsen/logrus"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/helper"

	"github.com/trustbloc/edge-service/pkg/doc/vc/profile"
	"github.com/trustbloc/edge-service/pkg/restapi/vc/operation"
	"github.com/trustbloc/edge-service/test/bdd/pkg/bddutil"
	"github.com/trustbloc/edge-service/test/bdd/pkg/context"
)

const (
	issuerURL   = "http://localhost:8070"
	sidetreeURL = "https://localhost:48326/document"

	issueCredentialURLFormat           = issuerURL + "/%s" + "/credentials/issueCredential"
	composeAndIssueCredentialURLFormat = issuerURL + "/%s" + "/credentials/composeAndIssueCredential"
)

const (
	sha2_256            = 18
	recoveryRevealValue = "recoveryOTP"
	updateRevealValue   = "updateOTP"
	pubKeyIndex1        = "#key-1"
	defaultKeyType      = "Ed25519VerificationKey2018"

	validContext = `"@context":["https://www.w3.org/2018/credentials/v1"]`
	validVC      = `{` +
		validContext + `,
	  "id": "http://example.edu/credentials/1872",
	  "type": "VerifiableCredential",
	  "credentialSubject": {
		"id": "did:example:ebfeb1f712ebc6f1c276e12ec21"
	  },
	  "issuer": {
		"id": "did:example:76e12ec712ebc6f1c221ebfeb1f",
		"name": "Example University"
	  },
	  "issuanceDate": "2010-01-01T19:23:24Z",
	  "credentialStatus": {
		"id": "https://example.gov/status/24",
		"type": "CredentialStatusList2017"
	  }
	}`

	composeCredReqFormat = `{
	   "issuer":"did:example:uoweu180928901",
	   "subject":"did:example:oleh394sqwnlk223823ln",
	   "types":[
		  "UniversityDegree"
	   ],
	   "issuanceDate":"2020-03-25T19:38:54.45546Z",
	   "expirationDate":"2020-06-25T19:38:54.45546Z",
	   "claims":{
		  "customField":"customFieldVal",
		  "name":"John Doe"
	   },
	   "evidence":{
		  "customField":"customFieldVal",
		  "id":"http://example.com/policies/credential/4",
		  "type":"IssuerPolicy"
	   },
	   "termsOfUse":{
		  "id":"http://example.com/policies/credential/4",
		  "type":"IssuerPolicy"
	   },
	   "proofFormat":"jws",
	   "proofFormatOptions":{
		  "kid":` + `"%s"` + `
	   }
	}`
)

// Steps is steps for VC BDD tests
type Steps struct {
	bddContext *context.BDDContext
}

// NewSteps returns new agent from client SDK
func NewSteps(ctx *context.BDDContext) *Steps {
	return &Steps{bddContext: ctx}
}

// RegisterSteps registers agent steps
func (e *Steps) RegisterSteps(s *godog.Suite) {
	s.Step(`^"([^"]*)" has stored her transcript from the University$`, e.createCredential)
	s.Step(`^"([^"]*)" has a DID with the public key generated from Issuer Service - Generate Keypair API$`, e.createDID)
	s.Step(`^"([^"]*)" creates an Issuer Service profile "([^"]*)" with the DID$`, e.createIssuerProfile)
	s.Step(`^"([^"]*)" application service verifies the credential created by Issuer Service - Issue Credential API with it's DID$`, //nolint: lll
		e.issueAndVerifyCredential)
	s.Step(`^"([^"]*)" application service verifies the credential created by Issuer Service - Compose And Issue Credential API with it's DID$`, //nolint: lll
		e.composeIssueAndVerifyCredential)
}

func (e *Steps) createDID(user string) error {
	publicKey, err := e.generateKeypair()
	if err != nil {
		return err
	}

	doc, err := e.createSidetreeDID(publicKey)
	if err != nil {
		return err
	}

	e.bddContext.Args[bddutil.GetDIDKey(user)] = doc.ID

	return bddutil.ResolveDID(e.bddContext.VDRI, doc.ID, 10)
}

func (e *Steps) generateKeypair() (string, error) {
	resp, err := http.Get(issuerURL + "/kms/generatekeypair") //nolint: bodyclose
	if err != nil {
		return "", err
	}

	defer bddutil.CloseResponseBody(resp.Body)

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", bddutil.ExpectedStatusCodeError(http.StatusOK, resp.StatusCode, respBytes)
	}

	generateKeyPairResponse := operation.GenerateKeyPairResponse{}

	err = json.Unmarshal(respBytes, &generateKeyPairResponse)
	if err != nil {
		return "", err
	}

	return generateKeyPairResponse.PublicKey, nil
}

func (e *Steps) createIssuerProfile(user, profileName string) error {
	profileRequest := operation.ProfileRequest{}

	err := json.Unmarshal(e.bddContext.ProfileRequestTemplate, &profileRequest)
	if err != nil {
		return err
	}

	userDID := e.bddContext.Args[bddutil.GetDIDKey(user)]

	profileRequest.Name = uuid.New().String() + profileName
	profileRequest.DID = userDID

	requestBytes, err := json.Marshal(profileRequest)
	if err != nil {
		return err
	}

	resp, err := http.Post(issuerURL+"/profile", "", bytes.NewBuffer(requestBytes)) //nolint: bodyclose
	if err != nil {
		return err
	}

	defer bddutil.CloseResponseBody(resp.Body)

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return bddutil.ExpectedStatusCodeError(http.StatusCreated, resp.StatusCode, respBytes)
	}

	profileResponse := profile.DataProfile{}

	err = json.Unmarshal(respBytes, &profileResponse)
	if err != nil {
		return err
	}

	if userDID != profileResponse.DID {
		return fmt.Errorf("DID not saved in the profile - expected=%s actual=%s", userDID, profileResponse.DID)
	}

	e.bddContext.Args[bddutil.GetProfileNameKey(user)] = profileResponse.Name

	return bddutil.ResolveDID(e.bddContext.VDRI, profileResponse.DID, 10)
}

func (e *Steps) createSidetreeDID(base58PubKey string) (*docdid.Doc, error) {
	req, err := e.buildSideTreeRequest(base58PubKey)
	if err != nil {
		return nil, err
	}

	return e.sendCreateRequest(req)
}

func (e *Steps) verifyCredential(signedVCByte []byte) error {
	signedVCResp := make(map[string]interface{})

	err := json.Unmarshal(signedVCByte, &signedVCResp)
	if err != nil {
		return err
	}

	proof, ok := signedVCResp["proof"].(map[string]interface{})
	if !ok {
		return errors.New("unable to convert proof to a map")
	}

	if proof["type"] != "Ed25519Signature2018" {
		return errors.New("proof type is not valid")
	}

	if proof["jws"] == "" {
		return errors.New("proof jws value is empty")
	}

	return nil
}

func (e *Steps) issueCredential(user, did string) ([]byte, error) {
	if err := bddutil.ResolveDID(e.bddContext.VDRI, did, 10); err != nil {
		return nil, err
	}

	req := &operation.IssueCredentialRequest{
		Credential: []byte(validVC),
		Opts:       &operation.IssueCredentialOptions{AssertionMethod: did},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	endpointURL := fmt.Sprintf(issueCredentialURLFormat, e.bddContext.Args[bddutil.GetProfileNameKey(user)])

	resp, err := http.Post(endpointURL, "application/json", bytes.NewBuffer(reqBytes)) //nolint
	if err != nil {
		return nil, err
	}

	defer bddutil.CloseResponseBody(resp.Body)

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response : %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got unexpected response from %s status '%d' body %s",
			endpointURL, resp.StatusCode, responseBytes)
	}

	return responseBytes, nil
}

func (e *Steps) issueAndVerifyCredential(user string) error {
	did := e.bddContext.Args[bddutil.GetDIDKey(user)]
	log.Infof("DID for signing %s", did)

	signedVCByte, err := e.issueCredential(user, did)
	if err != nil {
		return err
	}

	return e.verifyCredential(signedVCByte)
}

func (e *Steps) composeIssueAndVerifyCredential(user string) error {
	did := e.bddContext.Args[bddutil.GetDIDKey(user)]
	log.Infof("DID for signing %s", did)

	if err := bddutil.ResolveDID(e.bddContext.VDRI, did, 10); err != nil {
		return err
	}

	req := fmt.Sprintf(composeCredReqFormat, did+pubKeyIndex1)

	endpointURL := fmt.Sprintf(composeAndIssueCredentialURLFormat, e.bddContext.Args[bddutil.GetProfileNameKey(user)])

	resp, err := http.Post(endpointURL, "application/json", bytes.NewBufferString(req)) //nolint
	if err != nil {
		return err
	}

	defer bddutil.CloseResponseBody(resp.Body)

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response : %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got unexpected response from %s status '%d' body %s",
			endpointURL, resp.StatusCode, responseBytes)
	}

	return e.verifyCredential(responseBytes)
}

func (e *Steps) createCredential(user string) error {
	if err := e.createDID(user); err != nil {
		return err
	}

	if err := e.createIssuerProfile(user, uuid.New().String()); err != nil {
		return err
	}

	signedVCByte, err := e.issueCredential(user, e.bddContext.Args[bddutil.GetDIDKey(user)])
	if err != nil {
		return err
	}

	if err := e.verifyCredential(signedVCByte); err != nil {
		return err
	}

	e.bddContext.Args[bddutil.GetCredentialKey(user)] = string(signedVCByte)

	return nil
}

func (e *Steps) buildSideTreeRequest(base58PubKey string) ([]byte, error) {
	publicKey := docdid.PublicKey{
		ID:    pubKeyIndex1,
		Type:  defaultKeyType,
		Value: base58.Decode(base58PubKey),
	}

	t := time.Now()

	didDoc := &docdid.Doc{
		Context:   []string{},
		PublicKey: []docdid.PublicKey{publicKey},
		Created:   &t,
		Updated:   &t,
	}

	docBytes, err := didDoc.JSONBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get document bytes : %s", err)
	}

	req, err := helper.NewCreateRequest(&helper.CreateRequestInfo{
		OpaqueDocument:          string(docBytes),
		RecoveryKey:             "recoveryKey",
		NextRecoveryRevealValue: []byte(recoveryRevealValue),
		NextUpdateRevealValue:   []byte(updateRevealValue),
		MultihashCode:           sha2_256,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sidetree request: %w", err)
	}

	return req, nil
}

func (e *Steps) sendCreateRequest(req []byte) (*docdid.Doc, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: e.bddContext.TLSConfig,
		}}

	resp, err := client.Post(sidetreeURL, "application/json", bytes.NewBuffer(req)) //nolint: bodyclose
	if err != nil {
		return nil, err
	}

	defer bddutil.CloseResponseBody(resp.Body)

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response : %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got unexpected response from %s status '%d' body %s",
			sidetreeURL, resp.StatusCode, responseBytes)
	}

	didDoc, err := docdid.ParseDocument(responseBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public DID document: %s", err)
	}

	return didDoc, nil
}