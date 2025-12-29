package protocol

import "testing"

func TestValidateToolFailures(t *testing.T) {
	cases := []struct {
		name string
		tool Tool
	}{
		{name: "missing tenant", tool: Tool{Name: "echo", Version: "1.0", InputSchema: []byte("{}"), OutputSchema: []byte("{}")}},
		{name: "empty name", tool: Tool{TenantID: "t1", Name: "", Version: "1.0", InputSchema: []byte("{}"), OutputSchema: []byte("{}")}},
		{name: "name with space", tool: Tool{TenantID: "t1", Name: "bad name", Version: "1.0", InputSchema: []byte("{}"), OutputSchema: []byte("{}")}},
		{name: "missing input schema", tool: Tool{TenantID: "t1", Name: "echo", Version: "1.0", OutputSchema: []byte("{}")}},
		{name: "missing output schema", tool: Tool{TenantID: "t1", Name: "echo", Version: "1.0", InputSchema: []byte("{}")}},
	}
	for _, tc := range cases {
		if err := ValidateTool(tc.tool); err == nil {
			t.Fatalf("expected error for case %s", tc.name)
		}
	}
}

func TestValidateInvocationFailures(t *testing.T) {
	badVersion := InvocationRequest{Version: "v2", TenantID: "t1", Tool: ToolRef{Name: "echo"}, Input: []byte(`{"x":1}`)}
	if err := ValidateInvocation(badVersion); err == nil {
		t.Fatalf("expected error for bad version")
	}

	missingTenant := InvocationRequest{Version: CurrentVersion, Tool: ToolRef{Name: "echo"}, Input: []byte(`{"x":1}`)}
	if err := ValidateInvocation(missingTenant); err == nil {
		t.Fatalf("expected error for missing tenant")
	}

	missingTool := InvocationRequest{Version: CurrentVersion, TenantID: "t1", Input: []byte(`{"x":1}`)}
	if err := ValidateInvocation(missingTool); err == nil {
		t.Fatalf("expected error for missing tool")
	}

	missingInput := InvocationRequest{Version: CurrentVersion, TenantID: "t1", Tool: ToolRef{Name: "echo"}}
	if err := ValidateInvocation(missingInput); err == nil {
		t.Fatalf("expected error for missing input")
	}
}
