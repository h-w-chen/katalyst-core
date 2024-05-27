package msr

import "testing"

func TestMSRDev_Write(t *testing.T) {
	t.Parallel()

	// set up test stub
	SetupTestSyscaller()

	type fields struct {
		fd int
	}
	type args struct {
		regno int64
		value uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				fd: 3,
			},
			args: args{
				regno: 7,
				value: 123,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := MSRDev{
				fd: tt.fields.fd,
			}
			if err := d.Write(tt.args.regno, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteMSR(t *testing.T) {
	t.Parallel()

	// set up test stub
	SetupTestSyscaller()

	type args struct {
		cpu   uint32
		msr   int64
		value uint64
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				cpu:   1,
				msr:   156,
				value: 123,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := WriteMSR(tt.args.cpu, tt.args.msr, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("WriteMSR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
