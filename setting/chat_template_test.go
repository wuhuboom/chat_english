package setting

import "testing"

func TestValidateH5ChatTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "modern template", value: "chat_page.html", want: "chat_page.html"},
		{name: "classic one without time", value: "chat_page1_notime.html", want: "chat_page1_notime.html"},
		{name: "classic three without time", value: "chat_page3_notime.html", want: "chat_page3_notime.html"},
		{name: "legacy template", value: " chat_page3-1.html ", want: "chat_page3-1.html"},
		{name: "wechat style template", value: "chat_wx.html", want: "chat_wx.html"},
		{name: "unknown template", value: "unknown.html", wantErr: true},
		{name: "path traversal", value: "../login.html", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateH5ChatTemplate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateH5ChatTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ValidateH5ChatTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveH5ChatTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		databaseValue string
		fileValue     string
		want          string
	}{
		{
			name:          "database wins",
			databaseValue: "chat_page5.html",
			fileValue:     "chat_page3.html",
			want:          "chat_page5.html",
		},
		{
			name:      "legacy file fallback",
			fileValue: "chat_page4.html",
			want:      "chat_page4.html",
		},
		{
			name:          "invalid database uses valid file",
			databaseValue: "../login.html",
			fileValue:     "chat_page1.html",
			want:          "chat_page1.html",
		},
		{
			name:          "invalid values use default",
			databaseValue: "missing.html",
			fileValue:     "also-missing.html",
			want:          DefaultH5ChatTemplate,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveH5ChatTemplate(tt.databaseValue, tt.fileValue); got != tt.want {
				t.Fatalf("ResolveH5ChatTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveEntH5ChatTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		entValue      string
		databaseValue string
		fileValue     string
		want          string
	}{
		{name: "enterprise override wins", entValue: "chat_page4.html", databaseValue: "chat_page5.html", fileValue: "chat_page1.html", want: "chat_page4.html"},
		{name: "invalid enterprise uses database", entValue: "../bad.html", databaseValue: "chat_page5.html", fileValue: "chat_page1.html", want: "chat_page5.html"},
		{name: "empty enterprise uses file fallback", fileValue: "chat_page3.html", want: "chat_page3.html"},
		{name: "all invalid uses modern default", entValue: "bad", databaseValue: "bad", fileValue: "bad", want: DefaultH5ChatTemplate},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ResolveEntH5ChatTemplate(tt.entValue, tt.databaseValue, tt.fileValue); got != tt.want {
				t.Fatalf("ResolveEntH5ChatTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}
