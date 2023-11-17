package mock

import (
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/harbor"
	"github.com/khulnasoft-lab/harbor-scanner-tunnel/pkg/tunnel"
	"github.com/stretchr/testify/mock"
)

type Transformer struct {
	mock.Mock
}

func NewTransformer() *Transformer {
	return &Transformer{}
}

func (t *Transformer) Transform(artifact harbor.Artifact, source []tunnel.Vulnerability) harbor.ScanReport {
	args := t.Called(artifact, source)
	return args.Get(0).(harbor.ScanReport)
}
