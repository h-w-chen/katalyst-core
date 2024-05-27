package rdt

import (
	"testing"

	"github.com/kubewharf/katalyst-core/pkg/util/mbw/msr"
)

func TestGetRDTValue(t *testing.T) {
	t.Parallel()

	// set up test stub
	msr.SetupTestSyscaller()

	type args struct {
		core  uint32
		event PQOS_EVENT_TYPE
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "negtive path returns error",
			args: args{
				core:  2,
				event: 11,
			},
			want:    0xFFFFFFFF,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetRDTValue(tt.args.core, tt.args.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRDTValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetRDTValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}
