package query2

import (
	"reflect"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
)

func Test_readEntity(t *testing.T) {
	type args struct {
		ad     appdef.IAppDef
		record istructs.IRecord
	}
	tests := []struct {
		name       string
		args       args
		wantEntity map[string]interface{}
		wantErr    bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEntity, err := readEntity(tt.args.ad, tt.args.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("readEntity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotEntity, tt.wantEntity) {
				t.Errorf("readEntity() gotEntity = %v, want %v", gotEntity, tt.wantEntity)
			}
		})
	}
}
