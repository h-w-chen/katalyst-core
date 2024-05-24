package mbw

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/afero"
)

func TestContains(t *testing.T) {
	t.Parallel()
	type args struct {
		array []int
		item  int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "happy path",
			args: args{array: []int{1, 2}, item: 2},
			want: true,
		},
		{
			name: "negative path",
			args: args{array: []int{1, 2}, item: 3},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Contains(tt.args.array, tt.args.item); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDelta(t *testing.T) {
	t.Parallel()
	type args struct {
		bit int
		new uint64
		old uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "happy path of 64",
			args: args{
				bit: 64,
				new: 22,
				old: 11,
			},
			want: 10,
		},
		{
			name: "happy path of 16",
			args: args{
				bit: 16,
				new: 22,
				old: 11,
			},
			want: 11,
		},
		{
			name: "special path of init 0",
			args: args{
				bit: 16,
				new: 22,
				old: 0,
			},
			want: 0,
		},
		{
			name: "round path of 64",
			args: args{
				bit: 64,
				new: 11,
				old: 22,
			},
			want: 18446744073709551604,
		},
		{
			name: "round path of 32",
			args: args{
				bit: 32,
				new: 11,
				old: 22,
			},
			want: 4294967284,
		},
		{
			name: "round path of 48",
			args: args{
				bit: 48,
				new: 11,
				old: 22,
			},
			want: 281474976710644,
		},
		{
			name: "rare 100",
			args: args{
				bit: 100,
				new: 11,
				old: 22,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Delta(tt.args.bit, tt.args.new, tt.args.old); got != tt.want {
				t.Errorf("Delta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRDTEventToMB(t *testing.T) {
	t.Parallel()
	type args struct {
		event    uint64
		interval uint64
		scalar   uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "happy path",
			args: args{
				event:    1024,
				interval: 1000,
				scalar:   1024,
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RDTEventToMB(tt.args.event, tt.args.interval, tt.args.scalar); got != tt.want {
				t.Errorf("RDTEventToMB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPMUToMB(t *testing.T) {
	t.Parallel()
	type args struct {
		count    uint64
		interval uint64
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "happy path",
			args: args{
				count:    1024 * 1024,
				interval: 1000,
			},
			want: 64,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := PMUToMB(tt.args.count, tt.args.interval); got != tt.want {
				t.Errorf("PMUToMB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBytesToGB(t *testing.T) {
	t.Parallel()
	if got := BytesToGB(1024 * 1024 * 1024); got != 1 {
		t.Errorf("expected 1; got %d", got)
	}
}

func TestGetCCDTopology(t *testing.T) {
	t.Parallel()

	// we would like to have below device files exist for testing
	fakeFiles := []struct {
		dir     string
		file    string
		content string
	}{
		{
			dir:     "/sys/devices/system/node/node0/cpu0/cache/index3/",
			file:    "shared_cpu_list",
			content: "0-1\n",
		},
		{
			dir:     "/sys/devices/system/node/node0/cpu1/cache/index3/",
			file:    "shared_cpu_list",
			content: "0-1\n",
		},
		{
			dir:     "/sys/devices/system/node/node1/cpu2/cache/index3/",
			file:    "shared_cpu_list",
			content: "2-3\n",
		},
		{
			dir:     "/sys/devices/system/node/node1/cpu3/cache/index3/",
			file:    "shared_cpu_list",
			content: "2-3\n",
		},
	}

	// set up fake fs replacing the source package-scoped  var
	appOS = &afero.Afero{Fs: afero.NewMemMapFs()}
	for _, entry := range fakeFiles {
		appOS.MkdirAll(entry.dir, os.ModePerm)
		appOS.WriteFile(filepath.Join(entry.dir, entry.file), []byte(entry.content), os.ModePerm)
	}

	type args struct {
		numNuma int
	}
	tests := []struct {
		name    string
		args    args
		want    map[int][]int
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				numNuma: 2,
			},
			want:    map[int][]int{0: {0, 1}, 1: {2, 3}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCCDTopology(tt.args.numNuma)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCCDTopology() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCCDTopology() got = %v, want %v", got, tt.want)
			}
		})
	}
}
