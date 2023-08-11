package cos

import (
	"testing"
)

// TestEncodeURIComponent tests 'EncodeURIComponent'.
func TestEncodeURIComponent(t *testing.T) {
	exclude := []byte("/")
	type args struct {
		s        string
		lower    bool
		excluded [][]byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"has -", args{"test-a.txt", true, [][]byte{exclude}}, "test-a.txt"},
		{"normal", args{"/txt/Test01-a.txt", true, [][]byte{exclude}}, "/txt/Test01-a.txt"},
		{"empty", args{"", true, [][]byte{exclude}}, ""},
		{"has space", args{"/test space.txt", true, [][]byte{exclude}}, "/test%20space.txt"},
		{"has chinese", args{"/mydoc/新建文件.txt", true, [][]byte{exclude}}, "/mydoc/%e6%96%b0%e5%bb%ba%e6%96%87%e4%bb%b6.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EncodeURIComponent(tt.args.s, tt.args.lower, tt.args.excluded...); got != tt.want {
				t.Errorf("EncodeURIComponent() = %v, want %v", got, tt.want)
			}
		})
	}
}
