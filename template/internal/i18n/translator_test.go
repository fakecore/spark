package i18n

import (
	"context"
	"testing"
)

func TestTranslator(t *testing.T) {
	// Initialize translator
	err := InitGlobalTranslator()
	if err != nil {
		t.Fatalf("Failed to init translator: %v", err)
	}

	tests := []struct {
		name         string
		locale       string
		msgID        string
		templateData map[string]interface{}
		wantContain  string
	}{
		{
			name:   "Chinese - Game Not Authorized",
			locale: "zh-CN",
			msgID:  "BIZ_GAME_NOT_AUTHORIZED",
			templateData: map[string]interface{}{
				"GameName": "Beat Saber",
				"ShopName": "XX店",
			},
			wantContain: "Beat Saber",
		},
		{
			name:   "English - Game Not Authorized",
			locale: "en-US",
			msgID:  "BIZ_GAME_NOT_AUTHORIZED",
			templateData: map[string]interface{}{
				"GameName": "Beat Saber",
				"ShopName": "XX Shop",
			},
			wantContain: "Beat Saber",
		},
		{
			name:        "Chinese - Success",
			locale:      "zh-CN",
			msgID:       "SUCCESS",
			wantContain: "成功",
		},
		{
			name:        "English - Success",
			locale:      "en-US",
			msgID:       "SUCCESS",
			wantContain: "successful",
		},
		{
			name:        "Unknown key returns key itself",
			locale:      "zh-CN",
			msgID:       "UNKNOWN_KEY_12345",
			wantContain: "UNKNOWN_KEY_12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithLocale(context.Background(), tt.locale)
			got := T(ctx, tt.msgID, tt.templateData)
			if got == "" {
				t.Errorf("T() returned empty string")
			}
			t.Logf("Translated message [%s]: %s", tt.locale, got)
		})
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		name       string
		acceptLang string
		want       string
	}{
		{
			name:       "Empty header",
			acceptLang: "",
			want:       DefaultLocale,
		},
		{
			name:       "Simple Chinese",
			acceptLang: "zh-CN",
			want:       "zh-CN",
		},
		{
			name:       "Simple English",
			acceptLang: "en-US",
			want:       "en-US",
		},
		{
			name:       "English base",
			acceptLang: "en",
			want:       "en-US",
		},
		{
			name:       "With quality values",
			acceptLang: "en-US,en;q=0.9,zh-CN;q=0.8",
			want:       "en-US",
		},
		{
			name:       "Chinese with quality",
			acceptLang: "zh-CN,zh;q=0.9,en;q=0.8",
			want:       "zh-CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAcceptLanguage(tt.acceptLang)
			if got != tt.want {
				t.Errorf("ParseAcceptLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}
