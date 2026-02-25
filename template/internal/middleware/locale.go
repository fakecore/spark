package middleware

import (
	"context"
	"strings"

	"spark/internal/i18n"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Locale extracts Accept-Language header and injects locale into context
func Locale() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Extract locale from Accept-Language header
			locale := extractLocale(ctx)

			// Inject locale into context
			ctx = i18n.WithLocale(ctx, locale)

			return handler(ctx, req)
		}
	}
}

// extractLocale extracts the locale from the Accept-Language header
func extractLocale(ctx context.Context) string {
	if info, ok := transport.FromServerContext(ctx); ok {
		acceptLang := info.RequestHeader().Get("Accept-Language")
		return parseAcceptLanguage(acceptLang)
	}
	return i18n.DefaultLocale
}

// parseAcceptLanguage parses the Accept-Language header and returns the best matching locale
// It handles quality values (q-parameters) and returns the first supported language
func parseAcceptLanguage(acceptLang string) string {
	if acceptLang == "" {
		return i18n.DefaultLocale
	}

	// Split by comma to get individual language entries
	entries := strings.Split(acceptLang, ",")

	// Parse and find the best match (first supported language)
	for _, entry := range entries {
		// Remove quality value if present (e.g., "en;q=0.9" -> "en")
		if idx := strings.Index(entry, ";"); idx != -1 {
			entry = entry[:idx]
		}

		// Trim whitespace
		entry = strings.TrimSpace(entry)

		// Map to supported locale
		locale := mapLanguageToLocale(entry)
		if locale != "" {
			return locale
		}
	}

	return i18n.DefaultLocale
}

// mapLanguageToLocale maps a language tag to a supported locale
func mapLanguageToLocale(lang string) string {
	// Normalize the input
	lang = strings.ToLower(strings.TrimSpace(lang))

	switch lang {
	case "en", "en-us", "en-gb", "en-ca", "en-au":
		return "en-US"
	case "zh", "zh-cn", "zh-hans", "zh-sg":
		return "zh-CN"
	case "zh-tw", "zh-hk", "zh-hant":
		return "zh-CN" // Traditional Chinese also maps to zh-CN for now
	default:
		return ""
	}
}
