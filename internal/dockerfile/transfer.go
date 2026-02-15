package dockerfile

import (
	"context"
	"io"
	"os"

	"github.com/bonus2k/contain-sentry/cmd/containsentry/config"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"go.uber.org/zap"
)

type Dockerfile struct {
	context *DockerStageEval
	nodes   []*parser.Node
	ctx     context.Context
}

func NewDockerfile(ctx context.Context, path string) (*Dockerfile, error) {
	logger := config.FromCtx(ctx)

	file, err := os.Open(path)
	if err != nil {
		logger.Error("Error opening Dockerfile", zap.Error(err))
		return nil, err
	}
	defer func(file *os.File) {
		ec := file.Close()
		if ec != nil {
			logger.Error("Error closing file", zap.Error(ec))
		}
	}(file)

	var r io.Reader = file
	parse, err := parser.Parse(r)
	if err != nil {
		logger.Error("Error parsing Dockerfile", zap.Error(err))
		return nil, err
	}

	return &Dockerfile{
		context: &DockerStageEval{},
		nodes:   parse.AST.Children,
		ctx:     ctx,
	}, nil
}
