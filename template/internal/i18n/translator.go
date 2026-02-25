package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

// contextKey is the key type for context values
type contextKey string

const (
	// LocaleContextKey is the key for locale in context
	LocaleContextKey contextKey = "i18n-locale"
	// DefaultLocale is the fallback locale
	DefaultLocale = "zh-CN"
)

// Translator handles internationalization
type Translator struct {
	bundle *i18n.Bundle
}

// NewTranslator creates a new translator instance
func NewTranslator() (*Translator, error) {
	bundle := i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load embedded locale files
	files := []string{"zh-CN.json", "en-US.json"}
	for _, file := range files {
		data, err := localeFS.ReadFile("locales/" + file)
		if err != nil {
			return nil, fmt.Errorf("failed to read locale file %s: %w", file, err)
		}
		if _, err := bundle.ParseMessageFileBytes(data, file); err != nil {
			return nil, fmt.Errorf("failed to load locale file %s: %w", file, err)
		}
	}

	return &Translator{bundle: bundle}, nil
}

// Localize translates a message key to the target language
func (t *Translator) Localize(ctx context.Context, msgID string, templateData map[string]interface{}) string {
	locale := LocaleFromContext(ctx)

	localizer := i18n.NewLocalizer(t.bundle, locale, DefaultLocale)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: templateData,
	})
	if err != nil {
		// Fallback to message ID if translation fails
		return msgID
	}

	return msg
}

// LocalizeWithDefault translates a message with a default value if translation not found
func (t *Translator) LocalizeWithDefault(ctx context.Context, msgID string, defaultMsg string, templateData map[string]interface{}) string {
	locale := LocaleFromContext(ctx)

	localizer := i18n.NewLocalizer(t.bundle, locale, DefaultLocale)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:      msgID,
		DefaultMessage: &i18n.Message{Other: defaultMsg},
		TemplateData:   templateData,
	})
	if err != nil {
		return defaultMsg
	}

	return msg
}

// LocaleFromContext extracts locale from context
func LocaleFromContext(ctx context.Context) string {
	if locale, ok := ctx.Value(LocaleContextKey).(string); ok && locale != "" {
		return locale
	}
	return DefaultLocale
}

// WithLocale injects locale into context
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, LocaleContextKey, locale)
}

// ParseAcceptLanguage parses Accept-Language header and returns the best matching locale
// It handles quality values (q-parameters) like "en-US,en;q=0.9,zh-CN;q=0.8"
func ParseAcceptLanguage(acceptLang string) string {
	if acceptLang == "" {
		return DefaultLocale
	}

	// Split by comma to handle multiple languages with quality values
	// Example: "en-US,en;q=0.9,zh-CN;q=0.8"
	languages := strings.Split(acceptLang, ",")

	for _, lang := range languages {
		// Remove quality value if present (e.g., "en;q=0.9" -> "en")
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}
		lang = strings.TrimSpace(lang)

		if lang == "" {
			continue
		}

		// Parse the language tag
		tag, err := language.Parse(lang)
		if err != nil {
			continue
		}

		// Map to supported locales
		switch tag {
		case language.AmericanEnglish, language.English:
			return "en-US"
		case language.Chinese, language.SimplifiedChinese, language.TraditionalChinese:
			return "zh-CN"
		default:
			// Try to match by base language
			base, _ := tag.Base()
			switch base.String() {
			case "en":
				return "en-US"
			case "zh":
				return "zh-CN"
			}
		}
	}

	return DefaultLocale
}

// Global translator instance
var globalTranslator *Translator

// InitGlobalTranslator initializes the global translator
func InitGlobalTranslator() error {
	t, err := NewTranslator()
	if err != nil {
		return err
	}
	globalTranslator = t
	return nil
}

// T is a shortcut for translating messages using the global translator
func T(ctx context.Context, msgID string, templateData map[string]interface{}) string {
	if globalTranslator == nil {
		return msgID
	}
	return globalTranslator.Localize(ctx, msgID, templateData)
}

// TDefault translates with a default message
func TDefault(ctx context.Context, msgID string, defaultMsg string, templateData map[string]interface{}) string {
	if globalTranslator == nil {
		return defaultMsg
	}
	return globalTranslator.LocalizeWithDefault(ctx, msgID, defaultMsg, templateData)
}
