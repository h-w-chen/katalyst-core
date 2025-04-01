package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_isNumaExclusive(t *testing.T) {
	t.Parallel()
	type args struct {
		annotations map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "happy path of positive",
			args: args{
				annotations: map[string]string{
					"katalyst.kubewharf.io/memory_enhancement": `{"numa_binding": "true", "numa_exclusive": "true"}`,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tt.want, isNumaExclusive(tt.args.annotations), "isNumaExclusive(%v)", tt.args.annotations)
		})
	}
}
