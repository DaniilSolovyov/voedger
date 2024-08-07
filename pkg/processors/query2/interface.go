package query2

import (
	"context"

	"github.com/voedger/voedger/pkg/pipeline"
	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"
)

type RowsProcessorFactory func(ctx context.Context, rs IResultSenderClosable, sender ibus.ISender) (ap pipeline.IAsyncPipeline)
