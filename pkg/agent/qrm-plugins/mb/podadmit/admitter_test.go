package podadmit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cloneAugmentedAnnotation(t *testing.T) {
	t.Parallel()
	type args struct {
		qosLevel string
		anno     map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "happy path of batch job being shared_30",
			args: args{
				qosLevel: "shared_cores",
				anno: map[string]string{
					"foo":                                   "bar",
					"katalyst.kubewharf.io/cpu_enhancement": `{"cpuset_pool": "batch"}`,
				},
			},
			want: map[string]string{
				"rdt.resources.beta.kubernetes.io/pod":  "shared_30",
				"foo":                                   "bar",
				"katalyst.kubewharf.io/cpu_enhancement": `{"cpuset_pool": "batch"}`,
			},
		},
		{
			name: "default no augment",
			args: args{
				qosLevel: "shared_cores",
				anno: map[string]string{
					"foo": "bar",
				},
			},
			want: map[string]string{
				"foo": "bar",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, cloneAugmentedAnnotation(tt.args.qosLevel, tt.args.anno), "cloneAugmentedAnnotation(%v, %v)", tt.args.qosLevel, tt.args.anno)
		})
	}
}

func Test_isOfSocketPod(t *testing.T) {
	t.Parallel()
	type args struct {
		qosLevel string
		podRole  string
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
				podRole:  "socket-service",
			},
			want: true,
		},
		{
			name: "negative with others",
			args: args{
				qosLevel: "dedicated_cores",
				podRole:  "micro-service",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, isOfSocketPod(tt.args.qosLevel, tt.args.podRole), "isOfSocketPod(%v)", tt.args)
		})
	}
}
