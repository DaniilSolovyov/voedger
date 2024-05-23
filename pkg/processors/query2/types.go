/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"reflect"
)

type workpiece struct {
	ad          appdef.IAppDef
	records     istructs.IRecords
	viewRecords istructs.IViewRecords
	wsid        istructs.WSID
	id          istructs.RecordID
	name        appdef.QName
	entity      interface{}
	bb          []byte
}

type switchByRequestType struct{}

func (s switchByRequestType) Switch(work interface{}) (branchName string, err error) {
	w := work.(workpiece)
	switch w.ad.Type(w.name).Kind() {
	case appdef.TypeKind_WDoc, appdef.TypeKind_WRecord:
		return branch_DocByID, nil
	case appdef.TypeKind_ViewRecord:
		return branch_View, nil
	case appdef.TypeKind_Query:
		return branch_QueryFunction, nil
	case appdef.TypeKind_CDoc, appdef.TypeKind_CRecord:
		return branch_Collection, nil
	default:
		panic("impossible")
	}
}

type switchByEntityType struct{}

func (s switchByEntityType) Switch(work interface{}) (branchName string, err error) {
	w := work.(workpiece)
	if reflect.TypeOf(w.entity).Kind() == reflect.Array {
		return branch_Collection_Many, nil
	}
	return branch_Collection_One, nil
}
