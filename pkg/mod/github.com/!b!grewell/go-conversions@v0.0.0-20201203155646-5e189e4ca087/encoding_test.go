package conversions

import (
	"reflect"
	"testing"
)

func TestConvertToUTF16LEString(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name       string
		args       args
		wantOutput string
	}{
		// TODO: Add test cases.
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := ConvertToUTF16LEString(tt.args.input); gotOutput != tt.wantOutput {
				t.Errorf("ConvertToUTF16LEString() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}

func TestConvertToUTF16Slice(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name       string
		args       args
		wantOutput []uint16
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := ConvertToUTF16Slice(tt.args.input); !reflect.DeepEqual(gotOutput, tt.wantOutput) {
				t.Errorf("ConvertToUTF16Slice() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}

func TestConvertUTF16ToLEBytes(t *testing.T) {
	type args struct {
		input []uint16
	}
	tests := []struct {
		name       string
		args       args
		wantOutput []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := ConvertUTF16ToLEBytes(tt.args.input); !reflect.DeepEqual(gotOutput, tt.wantOutput) {
				t.Errorf("ConvertUTF16ToLEBytes() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}

func TestConvertToUTF16LEBase64String(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name       string
		args       args
		wantOutput string
	}{
		{
			name: "basic encoding",
			args: args{
				input: "this is a test",
			},
			wantOutput: "dABoAGkAcwAgAGkAcwAgAGEAIAB0AGUAcwB0AA==",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := ConvertToUTF16LEBase64String(tt.args.input); gotOutput != tt.wantOutput {
				t.Errorf("ConvertToUTF16LEBase64String() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}