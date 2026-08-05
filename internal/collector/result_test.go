package collector

import "testing"

func TestResultValidation(t *testing.T) {
	tests := []struct {
		name    string
		result  Result
		wantErr bool
	}{
		{name: "success", result: Success("snapshot")},
		{name: "partial", result: Partial("partial snapshot", "field_unavailable")},
		{name: "failure", result: Failure("collector_error")},
		{name: "failure without code", result: Result{Status: "failed"}, wantErr: true},
		{name: "success with code", result: Result{Status: "succeeded", ErrorCode: "unexpected"}, wantErr: true},
		{name: "invalid code", result: Failure("permission denied: /secret"), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.result.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
