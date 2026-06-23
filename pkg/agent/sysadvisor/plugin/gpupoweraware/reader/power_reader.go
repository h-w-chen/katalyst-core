package reader

import "context"

type PowerReader interface {
	Start() error
	GetTotalPower(ctx context.Context) (int, error)
}

type gpuPowerReader struct {
}

func (g *gpuPowerReader) Start() error {
	//TODO implement me
	panic("implement me")
}

func (g *gpuPowerReader) GetTotalPower(ctx context.Context) (int, error) {
	//TODO implement me
	panic("implement me")
}

func New() PowerReader {
	return &gpuPowerReader{}
}
