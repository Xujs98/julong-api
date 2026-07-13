package console_setting

import "testing"

func TestValidateCustomEndpoints(t *testing.T) {
	valid := `[{"id":1,"name":"OpenAI Compatible","url":"https://api.example.com/v1","description":"Supports OpenAI-format requests"}]`
	if err := ValidateConsoleSettings(valid, "CustomEndpoints"); err != nil {
		t.Fatalf("valid endpoints rejected: %v", err)
	}
	invalid := `[{"id":1,"name":"Broken","url":"javascript:alert(1)","description":"unsafe"}]`
	if err := ValidateConsoleSettings(invalid, "CustomEndpoints"); err == nil {
		t.Fatal("invalid endpoint URL was accepted")
	}
}
