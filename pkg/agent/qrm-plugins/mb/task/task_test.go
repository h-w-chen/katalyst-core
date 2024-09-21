package task

import "testing"

func TestTask_GetResctrlCtrlGroup(t1 *testing.T) {
	t1.Parallel()
	type fields struct {
		QoSLevel QoSLevel
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				QoSLevel: "shared_cores",
			},
			want:    "/sys/fs/resctrl/shared",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := Task{
				QoSLevel: tt.fields.QoSLevel,
			}
			got, err := t.GetResctrlCtrlGroup()
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetResctrlCtrlGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("GetResctrlCtrlGroup() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTask_GetResctrlMonGroup(t1 *testing.T) {
	t1.Parallel()
	type fields struct {
		PodUID   string
		QoSLevel QoSLevel
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				PodUID:   "111-222-333",
				QoSLevel: "dedicated_cores",
			},
			want:    "/sys/fs/resctrl/dedicated/mon_groups/pod111-222-333",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t1.Run(tt.name, func(t1 *testing.T) {
			t1.Parallel()
			t := Task{
				PodUID:   tt.fields.PodUID,
				QoSLevel: tt.fields.QoSLevel,
			}
			got, err := t.GetResctrlMonGroup()
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetResctrlMonGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("GetResctrlMonGroup() got = %v, want %v", got, tt.want)
			}
		})
	}
}
