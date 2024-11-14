package config

import (
	"reflect"
	"testing"
	"time"
)

type TestStruct struct {
	Bool        bool          `env:"BOOL" default:"true"`
	Duration    time.Duration `env:"DURATION" default:"1s"`
	Float64     float64       `env:"FLOAT64" default:"1.1"`
	Int         int           `env:"INT" default:"1"`
	Int64       int64         `env:"INT64" default:"1"`
	String      string        `env:"STRING" default:"string"`
	Uint        uint          `env:"UINT" default:"1"`
	Uint64      uint64        `env:"UINT64" default:"1"`
	NoEnv       string        `default:"noenv"`
	NoDefault   string        `env:"NO_DEFAULT"`
	NoStructTag string
}

func TestNew(t *testing.T) {
	makeLookup := func(m map[string]string) func(string) (string, bool) {
		return func(key string) (string, bool) {
			v, ok := m[key]
			return v, ok
		}
	}
	type args struct {
		lookupenv func(string) (string, bool)
		args      []string
		c         *TestStruct
	}
	tests := []struct {
		name    string
		args    args
		want    *TestStruct
		wantErr bool
	}{
		{name: "Defaults", args: args{lookupenv: makeLookup(map[string]string{}), args: []string{"ConfigTestApp"}, c: &TestStruct{}}, want: &TestStruct{Bool: true, Duration: time.Second, Float64: 1.1, Int: 1, Int64: 1, String: "string", Uint: 1, Uint64: 1, NoEnv: "noenv", NoDefault: "", NoStructTag: ""}, wantErr: false},
		{name: "SetEnv", args: args{lookupenv: makeLookup(map[string]string{"BOOL": "false"}), args: []string{"ConfigTestApp"}, c: &TestStruct{}}, want: &TestStruct{Bool: false, Duration: time.Second, Float64: 1.1, Int: 1, Int64: 1, String: "string", Uint: 1, Uint64: 1, NoEnv: "noenv", NoDefault: "", NoStructTag: ""}, wantErr: false},
		{name: "SetArg", args: args{lookupenv: makeLookup(map[string]string{}), args: []string{"ConfigTestApp", "-BOOL=false"}, c: &TestStruct{}}, want: &TestStruct{Bool: false, Duration: time.Second, Float64: 1.1, Int: 1, Int64: 1, String: "string", Uint: 1, Uint64: 1, NoEnv: "noenv", NoDefault: "", NoStructTag: ""}, wantErr: false},
		{name: "SetEnvAndArg", args: args{lookupenv: makeLookup(map[string]string{"INT": "100"}), args: []string{"ConfigTestApp", "-INT=111"}, c: &TestStruct{}}, want: &TestStruct{Bool: true, Duration: time.Second, Float64: 1.1, Int: 111, Int64: 1, String: "string", Uint: 1, Uint64: 1, NoEnv: "noenv", NoDefault: "", NoStructTag: ""}, wantErr: false},
		{name: "NoDefault", args: args{lookupenv: makeLookup(map[string]string{"NO_DEFAULT": "test"}), args: []string{"ConfigTestApp"}, c: &TestStruct{}}, want: &TestStruct{Bool: true, Duration: time.Second, Float64: 1.1, Int: 1, Int64: 1, String: "string", Uint: 1, Uint64: 1, NoEnv: "noenv", NoDefault: "test", NoStructTag: ""}, wantErr: false},
		{name: "InvalidArg", args: args{lookupenv: makeLookup(map[string]string{}), args: []string{"ConfigTestApp", "-bah"}, c: &TestStruct{}}, want: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.lookupenv, tt.args.args, tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
