package trace

import (
	"os"
	"strconv"
	"strings"
)

// LoadConfigFromEnv loads trace configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := DefaultConfig()

	if serviceName := os.Getenv("OTEL_SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	if serviceVersion := os.Getenv("OTEL_SERVICE_VERSION"); serviceVersion != "" {
		config.ServiceVersion = serviceVersion
	}

	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
	}

	// Optional: map a non-standard OTEL_ENVIRONMENT into resource attributes.
	if environment := os.Getenv("OTEL_ENVIRONMENT"); environment != "" {
		if config.ResourceAttributes == nil {
			config.ResourceAttributes = make(map[string]string)
		}
		config.ResourceAttributes["deployment.environment"] = environment
	}

	// Standard OTEL_RESOURCE_ATTRIBUTES: "k=v,k2=v2"
	if rawAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); rawAttrs != "" {
		if config.ResourceAttributes == nil {
			config.ResourceAttributes = make(map[string]string)
		}
		for _, part := range strings.Split(rawAttrs, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])
			if k == "" {
				continue
			}
			config.ResourceAttributes[k] = v
		}
	}

	if samplingRate := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); samplingRate != "" {
		if rate, err := strconv.ParseFloat(samplingRate, 64); err == nil {
			config.SamplingRate = rate
		}
	}

	if enabled := os.Getenv("OTEL_TRACES_ENABLED"); enabled != "" {
		config.Enabled = enabled == "true"
	}

	return config
}
