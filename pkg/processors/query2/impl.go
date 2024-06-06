/*
 * Copyright (c) 2024-present unTill Pro, Ltd.
 *
 * @author Daniil Solovyov
 */

package query2

import (
	"context"
	"encoding/json"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/pipeline"
	"github.com/voedger/voedger/pkg/sys/collection"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
	"net/http"
)

func newProcessor(ctx context.Context) pipeline.ISyncPipeline {
	return pipeline.NewSyncPipeline(ctx, "Query Processor 2",
		pipelineFunc("auth", auth),
		pipeline.WireSyncOperator("switch by request type", pipeline.SwitchOperator(switchByRequestType{},
			pipeline.SwitchBranch(branch_DocByID, pipeline.NewSyncPipeline(ctx, "read one document",
				pipelineFunc("read doc by id", readDocById),
				pipelineFunc("serialize", serialize),
				pipelineFunc("send result", sendResult),
			)),
			pipeline.SwitchBranch(branch_QueryFunction, pipeline.NewSyncPipeline(ctx, "execute query function",
				pipelineFunc("serialize", serialize),
				pipelineFunc("send result", sendResult),
			)),
			pipeline.SwitchBranch(branch_Collection, pipeline.NewSyncPipeline(ctx, "read collection of documents",
				pipelineFunc("read collection", readCollection),
				pipeline.WireSyncOperator("switch by entity type", pipeline.SwitchOperator(switchByEntityType{},
					pipeline.SwitchBranch(branch_Collection_One, pipeline.NewSyncPipeline(ctx, "one document",
						pipelineFunc("serialize", serialize),
						pipelineFunc("send result", sendResult),
					)),
					pipeline.SwitchBranch(branch_Collection_Many, pipeline.NewSyncPipeline(ctx, "many documents",
						pipelineFunc("wrap", wrap),
						pipelineFunc("serialize", serialize),
						pipelineFunc("send result", sendResult),
					)),
				)),
			)),
			pipeline.SwitchBranch(branch_View, pipeline.NewSyncPipeline(ctx, "read view",
				pipelineFunc("serialize", serialize),
				pipelineFunc("send result", sendResult),
			)),
		)),
	)
}

func auth(ctx context.Context, w *workpiece) (err error) {
	//TODO
	return nil
}
func readCollection(ctx context.Context, w *workpiece) (err error) {
	kb := w.viewRecords.KeyBuilder(collection.QNameCollectionView)
	kb.PutInt32(collection.Field_PartKey, collection.PartitionKeyCollection)
	kb.PutQName(collection.Field_DocQName, w.name)
	atMostOne := w.id != istructs.NullRecordID
	if atMostOne {
		kb.PutRecordID(collection.Field_DocID, w.id)
	}
	entities := make([]map[string]interface{}, 0)
	cb := func(key istructs.IKey, value istructs.IValue) (err error) {
		entity, err := readEntity(w.ad, value.AsRecord(collection.Field_Record))
		if err != nil {
			return
		}
		entities = append(entities, entity)
		return err
	}
	if e := w.viewRecords.Read(ctx, w.wsid, kb, cb); e != nil {
		return e
	}
	if atMostOne {
		entities = buildTree(entities)
		if len(entities) > 0 {
			w.entity = entities[0]
		}
	} else {
		w.entity = buildTree(entities)
	}
	return
}
func readDocById(_ context.Context, w *workpiece) (err error) {
	record, err := w.records.Get(w.wsid, true, w.id)
	if err != nil {
		return
	}
	w.entity, err = readEntity(w.ad, record)
	if err != nil {
		return
	}
	return
}
func wrap(_ context.Context, w *workpiece) (err error) {
	type result struct {
		Entity interface{} `json:"results"`
	}
	w.entity = result{Entity: w.entity}
	return
}
func serialize(_ context.Context, w *workpiece) (err error) {
	w.bb, err = json.Marshal(w.entity)
	return
}
func sendResult(_ context.Context, w *workpiece) (err error) {
	w.sender.SendResponse(ibus.Response{
		ContentType: "application/json",
		StatusCode:  http.StatusOK,
		Data:        w.bb,
	})
	return nil
}
func pipelineFunc(name string, doSync func(ctx context.Context, w *workpiece) (err error)) *pipeline.WiredOperator {
	return pipeline.WireFunc(name, func(ctx context.Context, work interface{}) (err error) {
		return doSync(ctx, work.(*workpiece))
	})
}
func buildTree(in []map[string]interface{}) (out []map[string]interface{}) {
	//TODO insert child entities to parent by sys.ParentID and sys.Container
	return in
}
func readEntity(ad appdef.IAppDef, record istructs.IRecord) (entity map[string]interface{}, err error) {
	entity = make(map[string]interface{})
	for _, field := range ad.Record(record.QName()).Fields() {
		switch field.DataKind() {
		case appdef.DataKind_int32:
			entity[field.Name()] = record.AsInt32(field.Name())
		case appdef.DataKind_int64:
			entity[field.Name()] = record.AsInt64(field.Name())
		case appdef.DataKind_float32:
			entity[field.Name()] = record.AsFloat32(field.Name())
		case appdef.DataKind_float64:
			entity[field.Name()] = record.AsFloat64(field.Name())
		case appdef.DataKind_bytes:
			entity[field.Name()] = record.AsBytes(field.Name())
		case appdef.DataKind_string:
			entity[field.Name()] = record.AsString(field.Name())
		case appdef.DataKind_QName:
			entity[field.Name()] = record.AsQName(field.Name())
		case appdef.DataKind_bool:
			entity[field.Name()] = record.AsBool(field.Name())
		case appdef.DataKind_RecordID:
			entity[field.Name()] = record.AsRecordID(field.Name())
		default:
			panic("impossible")
		}
	}
	return
}
