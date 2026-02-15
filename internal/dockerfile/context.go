package dockerfile

import "github.com/moby/buildkit/frontend/dockerfile/instructions"

func instructionStage(index int, stg *instructions.Stage) DockerStageState {
	return DockerStageState{
		StageIndex: index,
		StageName:  stg.Name,
		BaseImage:  stg.BaseName,
		Env:        map[string]Tracked[AbsString]{},
		Args:       map[string]Tracked[AbsString]{},
	}
}

type DockerStageEval struct {
	Stages   []DockerStageState
	Final    DockerStageState
	hasFinal bool
	command  instructions.Command
}

func (d *DockerStageEval) startStage(stg DockerStageState) {
	if d.hasFinal {
		d.Stages = append(d.Stages, d.Final)
	}
	d.Final = stg
	d.hasFinal = true
}

func (d *DockerStageEval) ensureFinalized() {
	if !d.hasFinal {
		return
	}
	d.Stages = append(d.Stages, d.Final)
	d.hasFinal = false
}

type DockerStageState struct {
	StageIndex      int
	StageName       string
	BaseImage       string
	User            Tracked[AbsString]
	Workdir         Tracked[AbsString]
	Shell           Tracked[[]string]
	Entrypoint      Tracked[[]string]
	Cmd             Tracked[[]string]
	Env             map[string]Tracked[AbsString]
	Args            map[string]Tracked[AbsString]
	HasUser         bool
	HasCopyFrom     bool
	HasHealthcheck  bool
	HasBuildTooling bool
}

type Tracked[T any] struct {
	Val      T
	Location SourceRef
}

type SourceRef struct {
	Start Position
	End   Position
}

type Position struct {
	Line      int
	Character int
}

type AbsString struct {
	Kind  string
	Known string
	Expr  string
	Deps  []string
}
