package corestate

type Stage string

const (
	StageNotReady Stage = "init"
	StagePreInit  Stage = "pre-init"
	StagePostInit Stage = "post-init"
	StageReady    Stage = "event"
)

const (
	StringsNone string = "none"
)

func NewCorestate(o *CoreState) *CoreState {
	// TODO: create a convenient interface for creating a state
	// if !utils.IsFullyInitialized(o) {
	// 	return nil, fmt.Errorf("CoreState is not fully initialized")
	// }
	return o
}
