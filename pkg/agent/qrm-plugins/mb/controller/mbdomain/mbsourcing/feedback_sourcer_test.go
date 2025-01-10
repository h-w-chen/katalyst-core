package mbsourcing

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
)

type mockSourcer struct {
	mock.Mock
}

func (m *mockSourcer) AttributeIncomingMBToSources(domainTargets []DomainMBTargetSource) []int {
	args := m.Called(domainTargets)
	return args.Get(0).([]int)
}

func Test_feedbackSourcer_AttributeIncomingMBToSources(t *testing.T) {
	t.Parallel()

	testDomainTargets := []DomainMBTargetSource{
		{MBSource: 55_747},
		{MBSource: 59_222},
	}

	dummySourcer := new(mockSourcer)
	dummySourcer.On("AttributeIncomingMBToSources", testDomainTargets).Return(
		[]int{55_656, 0})

	type fields struct {
		innerSourcer          Sourcer
		previousOutgoingQuota []int
	}
	type args struct {
		domainTargets []DomainMBTargetSource
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []int
	}{
		{
			name: "initial no history",
			fields: fields{
				innerSourcer:          dummySourcer,
				previousOutgoingQuota: nil,
			},
			args: args{
				domainTargets: testDomainTargets,
			},
			want: []int{55_656, 0},
		},
		{
			name: "initial no history",
			fields: fields{
				innerSourcer:          dummySourcer,
				previousOutgoingQuota: []int{54122, 0},
			},
			args: args{
				domainTargets: testDomainTargets,
			},
			want: []int{54_033, 0},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &feedbackSourcer{
				innerSourcer:          tt.fields.innerSourcer,
				previousOutgoingQuota: tt.fields.previousOutgoingQuota,
			}
			assert.Equalf(t, tt.want, f.AttributeIncomingMBToSources(tt.args.domainTargets), "AttributeIncomingMBToSources(%v)", tt.args.domainTargets)
		})
	}
}
