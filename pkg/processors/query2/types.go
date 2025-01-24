/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

type QueryParams struct {
	Constraints *Constraints           `json:"constraints"`
	Argument    map[string]interface{} `json:"argument,omitempty"`
	WSID        istructs.WSID
	Entity      appdef.QName
}

type Constraints struct {
	Order   []string               `json:"order"`
	Limit   int                    `json:"limit"`
	Skip    int                    `json:"skip"`
	Include []string               `json:"include"`
	Keys    []string               `json:"keys"`
	Where   map[string]interface{} `json:"where"`
}
