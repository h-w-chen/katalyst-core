package plugin

import (
	"context"
	"net"
	"testing"
)

func Test_powerCapAdvisorPluginServer_Cap(t *testing.T) {
	type fields struct {
		lis        net.Listener
		grpcServer *grpc.Server
	}
	type args struct {
		ctx         context.Context
		targetWatts int
		currWatt    int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := powerCapAdvisorPluginServer{
				lis:        tt.fields.lis,
				grpcServer: tt.fields.grpcServer,
			}
			p.Cap(tt.args.ctx, tt.args.targetWatts, tt.args.currWatt)
		})
	}
}
