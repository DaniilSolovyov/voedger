/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	coreutils "github.com/voedger/voedger/pkg/utils"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

func implRowsProcessorFactory(ctx context.Context, rs IResultSenderClosable, sender ibus.ISender) (ap pipeline.IAsyncPipeline) {
	return pipeline.NewAsyncPipeline(ctx, "qp2",
		pipeline.WireAsyncFunc("read", read),
		pipeline.WireAsyncFunc("serialize", serialize),
		pipeline.WireAsyncOperator("send", &send{
			rs:     rs,
			sender: sender,
		}),
	)
}

type Qwp struct {
	WSID  istructs.WSID
	Name  appdef.QName
	Where map[string]interface{}
	VR    istructs.IViewRecords
	AD    appdef.IAppDef
	OO    []map[string]interface{}
	BB    []byte
}

func (q *Qwp) Release() {}
func read(ctx context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	w := work.(*Qwp)
	kb := w.VR.KeyBuilder(w.Name)
	for k, v := range w.Where {
		field := w.AD.View(w.Name).Key().Field(k)
		if field == nil {
			return nil, fmt.Errorf("field '%s' is not a part of key", k)
		}
		switch v.(type) {
		case float64:
			kb.PutNumber(k, v.(float64))
		case string:
			kb.PutChars(k, v.(string))
		case bool:
			kb.PutBool(k, v.(bool))
		default:
			return nil, fmt.Errorf("field '%s' has insupported value", k)
		}
	}
	err = w.VR.Read(ctx, w.WSID, kb, func(key istructs.IKey, value istructs.IValue) (err error) {
		o := make(map[string]interface{})
		for _, field := range w.AD.View(w.Name).Key().Fields() {
			o[field.Name()] = coreutils.ReadByKind(field.Name(), field.DataKind(), key)
		}
		for _, field := range w.AD.View(w.Name).Value().Fields() {
			o[field.Name()] = coreutils.ReadByKind(field.Name(), field.DataKind(), value)
		}
		w.OO = append(w.OO, o)
		return
	})
	return w, err
}
func serialize(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	w := work.(*Qwp)
	w.BB, err = json.Marshal(w.OO)
	logger.Info(string(w.BB))
	if err != nil {
		return
	}
	return w, nil
}

type IResultSenderClosable interface {
	StartArraySection(sectionType string, path []string)
	StartMapSection(sectionType string, path []string)
	ObjectSection(sectionType string, path []string, element interface{}) (err error)
	SendElement(name string, element interface{}) (err error)
	Close(err error)
}

type send struct {
	pipeline.AsyncNOOP
	rs          IResultSenderClosable
	sender      ibus.ISender
	initialized bool
}

func (s *send) DoAsync(_ context.Context, work pipeline.IWorkpiece) (outWork pipeline.IWorkpiece, err error) {
	if !s.initialized {
		s.rs.StartArraySection("", nil)
		s.initialized = true
	}
	return work, s.rs.SendElement("", work.(*Qwp).BB)
	//s.sender.SendResponse(ibus.Response{
	//	ContentType: "application/json",
	//	StatusCode:  200,
	//	Data:        work.(*Qwp).BB,
	//})
	//return work, nil
}
func (s *send) OnError(_ context.Context, err error) {
	s.rs.Close(err)
}
