// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/edge-service/test/bdd

replace github.com/trustbloc/edge-service => ../..

go 1.16

require (
	github.com/btcsuite/btcutil v1.0.2
	github.com/cucumber/godog v0.9.0
	github.com/fsouza/go-dockerclient v1.6.0
	github.com/go-openapi/runtime v0.19.26
	github.com/go-openapi/strfmt v0.20.0
	github.com/google/uuid v1.2.0
	github.com/hyperledger/aries-framework-go v0.1.7-0.20210517160459-a72f856f36b8
	github.com/hyperledger/aries-framework-go-ext/component/vdr/orb v0.1.0
	github.com/hyperledger/aries-framework-go-ext/component/vdr/sidetree v0.0.0-20210517171415-871dc45ae58d
	github.com/hyperledger/aries-framework-go/component/storageutil v0.0.0-20210510053848-903ac6748b72
	github.com/hyperledger/aries-framework-go/spi v0.0.0-20210510053848-903ac6748b72
	github.com/igor-pavlenko/httpsignatures-go v0.0.23
	github.com/tidwall/gjson v1.6.7
	github.com/trustbloc/edge-core v0.1.7-0.20210517172158-aa11a4f18737
	github.com/trustbloc/edge-service v0.0.0-00010101000000-000000000000
	github.com/trustbloc/edv v0.1.7-0.20210510134838-bdb20956d60b
	github.com/trustbloc/kms v0.1.7-0.20210510144722-4d909760f6bf
	gotest.tools/v3 v3.0.3 // indirect
)

// https://github.com/ory/dockertest/issues/208#issuecomment-686820414
replace golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6
