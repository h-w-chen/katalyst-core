package resctrl

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
)

type mockCCDReader struct {
	mock.Mock
}

func (m *mockCCDReader) ReadMB(MonGroup string, ccd int) (int, error) {
	args := m.Called(MonGroup, ccd)
	return args.Int(0), args.Error(1)
}

func Test_monGroupReader_ReadMB(t *testing.T) {
	t.Parallel()

	mockDieReader := new(mockCCDReader)
	mockDieReader.On("ReadMB", "/sys/fs/resctrl/dedicated/mon_groups/pod001", 3).Return(111, nil)

	type fields struct {
		ccdReader CCDMBReader
	}
	type args struct {
		monGroup string
		dies     []int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[int]int
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				ccdReader: mockDieReader,
			},
			args: args{
				monGroup: "/sys/fs/resctrl/dedicated/mon_groups/pod001",
				dies:     []int{3},
			},
			want:    map[int]int{3: 111},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := monGroupReader{
				ccdReader: tt.fields.ccdReader,
			}
			got, err := m.ReadMB(tt.args.monGroup, tt.args.dies)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadMB() got = %v, want %v", got, tt.want)
			}
		})
	}
}
