package podadmit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isOfSocketPod(t *testing.T) {
	t.Parallel()
	type args struct {
		qosLevel string
		anno     map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "happy path of socket service",
			args: args{
				qosLevel: "dedicated_cores",
				anno:     map[string]string{"instance-model": "c3.1x"},
			},
			want: true,
		},
		{
			name: "negative with others",
			args: args{
				qosLevel: "dedicated_cores",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, IsSocketPod(tt.args.qosLevel, tt.args.anno), "isOfSocketPod(%v)", tt.args)
		})
	}
}

func Test_isBatchPod(t *testing.T) {
	t.Parallel()
	type args struct {
		qosLevel string
		anno     map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "happy path of shared-30",
			args: args{
				qosLevel: "shared_cores",
				anno:     map[string]string{"katalyst.kubewharf.io/cpu_enhancement": `{"cpuset_pool": "shared-30"}`},
			},
			want: true,
		},
		{
			name: "default not shared-30",
			args: args{
				qosLevel: "shared_cores",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, IsBatchPod(tt.args.qosLevel, tt.args.anno), "IsBatchPod(%v, %v)", tt.args.qosLevel, tt.args.anno)
		})
	}
}
