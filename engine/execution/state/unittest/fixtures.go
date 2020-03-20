package unittest

import (
	"github.com/dapperlabs/flow-go/engine/execution"
	"github.com/dapperlabs/flow-go/engine/execution/state/bootstrap"
	"github.com/dapperlabs/flow-go/engine/execution/state/delta"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/module/mempool/entity"
	"github.com/dapperlabs/flow-go/utils/unittest"
)

func EmptyView() *delta.View {
	view := delta.NewView(func(key flow.RegisterID) (bytes []byte, e error) {
		return nil, nil
	})

	bootstrap.BootstrapView(view) //create genesis state

	return view.NewChild() //return new view
}

func StateViewFixture() *delta.View {
	return delta.NewView(func(key flow.RegisterID) (bytes []byte, err error) {
		return nil, nil
	})
}
func ComputationResultFixture(n int) *execution.ComputationResult {
	stateViews := make([]*delta.View, n)
	for i := 0; i < n; i++ {
		stateViews[i] = StateViewFixture()
	}
	return &execution.ComputationResult{
		ExecutableBlock: unittest.ExecutableBlockFixture(n),
		StateViews:      stateViews,
	}
}

func ComputationResultForBlockFixture(completeBlock *entity.ExecutableBlock) *execution.ComputationResult {
	n := len(completeBlock.CompleteCollections)
	stateViews := make([]*delta.View, n)
	for i := 0; i < n; i++ {
		stateViews[i] = StateViewFixture()
	}
	return &execution.ComputationResult{
		ExecutableBlock: completeBlock,
		StateViews:      stateViews,
	}
}
