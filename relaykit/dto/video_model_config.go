package dto

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

const SuanliGatewayProvider = "suanliai"
const SuanliVideoPluginKey = "suanliai-video"

// VideoModelConfig is channel-local. Nil values inherit the unified protocol;
// explicit false and zero values remain distinguishable from an unset limit.
type VideoModelConfig struct {
	Enabled               *bool    `json:"enabled,omitempty"`
	MinSeconds            *int     `json:"min_seconds,omitempty"`
	MaxSeconds            *int     `json:"max_seconds,omitempty"`
	DefaultSeconds        *int     `json:"default_seconds,omitempty"`
	SupportedSeconds      []int    `json:"supported_seconds,omitempty"`
	AspectRatios          []string `json:"aspect_ratios,omitempty"`
	Resolutions           []string `json:"resolutions,omitempty"`
	DefaultAspectRatio    string   `json:"default_aspect_ratio,omitempty"`
	DefaultResolution     string   `json:"default_resolution,omitempty"`
	SupportsImages        *bool    `json:"supports_images,omitempty"`
	SupportsVideos        *bool    `json:"supports_videos,omitempty"`
	SupportsAudios        *bool    `json:"supports_audios,omitempty"`
	SupportsFace          *bool    `json:"supports_face,omitempty"`
	SupportsGenerateAudio *bool    `json:"supports_generate_audio,omitempty"`
	MaxImages             *int     `json:"max_images,omitempty"`
	MaxVideos             *int     `json:"max_videos,omitempty"`
	MaxAudios             *int     `json:"max_audios,omitempty"`
	FaceDefault           *bool    `json:"face_default,omitempty"`
	TimeoutSeconds        *int     `json:"timeout_seconds,omitempty"`
	MaxConcurrency        *int     `json:"max_concurrency,omitempty"`
	BillingMode           string   `json:"billing_mode,omitempty"`
	UnitPrice             *float64 `json:"unit_price,omitempty"`
	UnitCost              *float64 `json:"unit_cost,omitempty"`
	CostCurrency          string   `json:"cost_currency,omitempty"`
}

func (s ChannelSettings) ValidateVideoModelConfigs(maxDuration int) error {
	if s.GatewayProvider != "" && s.GatewayProvider != SuanliGatewayProvider {
		return fmt.Errorf("unknown video gateway provider")
	}
	if len(s.VideoModelConfigs) > 1024 {
		return fmt.Errorf("video_model_configs must not exceed 1024 models")
	}
	if len(s.VideoModelConfigs) > 0 && s.GatewayProvider != SuanliGatewayProvider {
		return fmt.Errorf("video_model_configs requires a video gateway provider")
	}
	for model, c := range s.VideoModelConfigs {
		if strings.TrimSpace(model) != model || model == "" || len(model) > 255 {
			return fmt.Errorf("invalid video model name")
		}
		minSeconds, maxSeconds := 1, maxDuration
		if c.MinSeconds != nil {
			minSeconds = *c.MinSeconds
		}
		if c.MaxSeconds != nil {
			maxSeconds = *c.MaxSeconds
		}
		if minSeconds < 1 || maxSeconds > maxDuration || minSeconds > maxSeconds {
			return fmt.Errorf("video model %q has invalid duration limits", model)
		}
		if c.DefaultSeconds != nil && (*c.DefaultSeconds < minSeconds || *c.DefaultSeconds > maxSeconds) {
			return fmt.Errorf("video model %q has invalid default duration", model)
		}
		if len(c.SupportedSeconds) > 128 {
			return fmt.Errorf("video model %q has too many supported durations", model)
		}
		for _, seconds := range c.SupportedSeconds {
			if seconds < minSeconds || seconds > maxSeconds {
				return fmt.Errorf("video model %q has invalid supported duration", model)
			}
		}
		if c.DefaultSeconds != nil && len(c.SupportedSeconds) > 0 && !slices.Contains(c.SupportedSeconds, *c.DefaultSeconds) {
			return fmt.Errorf("video model %q default duration is not supported", model)
		}
		for _, values := range [][]string{c.AspectRatios, c.Resolutions} {
			if len(values) > 128 {
				return fmt.Errorf("video model %q has too many output options", model)
			}
			for _, value := range values {
				if value == "" || value != strings.TrimSpace(value) || len(value) > 64 {
					return fmt.Errorf("video model %q has invalid output option", model)
				}
			}
		}
		if c.DefaultAspectRatio != "" && len(c.AspectRatios) > 0 && !slices.Contains(c.AspectRatios, c.DefaultAspectRatio) {
			return fmt.Errorf("video model %q default aspect ratio is not supported", model)
		}
		if c.DefaultResolution != "" && len(c.Resolutions) > 0 && !slices.Contains(c.Resolutions, c.DefaultResolution) {
			return fmt.Errorf("video model %q default resolution is not supported", model)
		}
		for _, count := range []*int{c.MaxImages, c.MaxVideos, c.MaxAudios} {
			if count != nil && (*count < 0 || *count > 128) {
				return fmt.Errorf("video model %q has invalid reference material limit", model)
			}
		}
		if c.TimeoutSeconds != nil && (*c.TimeoutSeconds < 1 || *c.TimeoutSeconds > 1800) {
			return fmt.Errorf("video model %q timeout must be between 1 and 1800 seconds", model)
		}
		if c.MaxConcurrency != nil && (*c.MaxConcurrency < 0 || *c.MaxConcurrency > 10000) {
			return fmt.Errorf("video model %q has invalid concurrency limit", model)
		}
		switch c.BillingMode {
		case "", "inherit", "per_second", "per_task":
		default:
			return fmt.Errorf("video model %q has invalid billing mode", model)
		}
		for _, price := range []*float64{c.UnitPrice, c.UnitCost} {
			if price != nil && (!finite(*price) || *price < 0 || *price > 1000000) {
				return fmt.Errorf("video model %q has invalid unit price or cost", model)
			}
		}
		if (c.BillingMode == "per_second" || c.BillingMode == "per_task") && c.UnitPrice == nil {
			return fmt.Errorf("video model %q requires a unit price for the selected billing mode", model)
		}
		if c.CostCurrency != "" && c.CostCurrency != "USD" && c.CostCurrency != "CNY" {
			return fmt.Errorf("video model %q has invalid cost currency", model)
		}
	}
	return nil
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }
