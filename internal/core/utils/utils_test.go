package utils

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestFunc_SetEviron(t *testing.T) {
	tests := []struct {
		eviron []string
		envs   []string
		want   []string
	}{
		{
			[]string{"ENV1=1", "ENV2=2", "ENV3=4"},
			[]string{"ENV3=3"},
			[]string{"ENV1=1", "ENV2=2", "ENV3=3"},
		},
		{
			[]string{"ENV1=1", "ENV2=5", "ENV3=4"},
			[]string{"ENV2=2", "ENV3=3"},
			[]string{"ENV1=1", "ENV2=2", "ENV3=3"},
		},
		{
			[]string{"ENV1=1", "ENV2=2", "ENV3=3"},
			[]string{"ENV4=4"},
			[]string{"ENV1=1", "ENV2=2", "ENV3=3", "ENV4=4"},
		},
		{
			[]string{"ENV1=1", "ENV2=2", "ENV3=4"},
			[]string{"ENV3=2", "ENV3=3"},
			[]string{"ENV1=1", "ENV2=2", "ENV3=3"},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("in %q set new %q", tt.eviron, tt.envs), func(t *testing.T) {
			got := SetEviron(tt.eviron, tt.envs...)
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetEviron(%q, %q) = got %v; want %v", tt.eviron, tt.envs, got, tt.want)
			}
		})
	}
}
