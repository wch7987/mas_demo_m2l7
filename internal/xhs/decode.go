package xhs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// VisualList unmarshals either a JSON array or a string containing that array.
// Qwen tool-calling often stringifies nested arrays.
type VisualList []ImageVisualAnalysis

func (l *VisualList) UnmarshalJSON(data []byte) error {
	items, err := decodeObjectArray(data, visualFromMap)
	if err != nil {
		return fmt.Errorf("images_visual: %w", err)
	}
	*l = items
	return nil
}

// EditList unmarshals either a JSON array or a string containing that array.
type EditList []ImageEditPlan

func (l *EditList) UnmarshalJSON(data []byte) error {
	items, err := decodeObjectArray(data, editFromMap)
	if err != nil {
		return fmt.Errorf("images_edit_plan: %w", err)
	}
	*l = items
	return nil
}

// StringList unmarshals a JSON array, a stringified array, or a single string.
type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	items, err := decodeJSONArray(data)
	if err != nil {
		return err
	}
	*s = items
	return nil
}

func decodeJSONArray(data []byte) ([]string, error) {
	data = unwrapJSON(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	if data[0] == '[' {
		var out []string
		if err := json.Unmarshal(data, &out); err == nil {
			return out, nil
		}
		var anyArr []any
		if err := json.Unmarshal(data, &anyArr); err != nil {
			return nil, err
		}
		out = make([]string, 0, len(anyArr))
		for _, v := range anyArr {
			out = append(out, anyToString(v))
		}
		return out, nil
	}
	if data[0] == '"' {
		var one string
		if err := json.Unmarshal(data, &one); err != nil {
			return nil, err
		}
		one = strings.TrimSpace(one)
		if strings.HasPrefix(one, "[") {
			return decodeJSONArray([]byte(one))
		}
		if one == "" {
			return nil, nil
		}
		return []string{one}, nil
	}
	return nil, fmt.Errorf("expected JSON array, got %s", truncateRunes(string(data), 80))
}

func decodeObjectArray[T any](data []byte, fromMap func(map[string]any) T) ([]T, error) {
	data = unwrapJSON(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		return decodeObjectArray([]byte(s), fromMap)
	}
	if data[0] == '{' {
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		return []T{fromMap(m)}, nil
	}
	if data[0] != '[' {
		return nil, fmt.Errorf("expected JSON array, got %s", truncateRunes(string(data), 80))
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]T, 0, len(raw))
	for _, item := range raw {
		item = unwrapJSON(item)
		if len(item) > 0 && item[0] == '"' {
			var s string
			if err := json.Unmarshal(item, &s); err != nil {
				return nil, err
			}
			item = unwrapJSON([]byte(s))
		}
		var m map[string]any
		if err := json.Unmarshal(item, &m); err != nil {
			return nil, err
		}
		out = append(out, fromMap(m))
	}
	return out, nil
}

func unwrapJSON(data []byte) []byte {
	data = bytes.TrimSpace(data)
	for len(data) >= 2 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			break
		}
		s = strings.TrimSpace(s)
		if s == "" || (s[0] != '[' && s[0] != '{' && s[0] != '"') {
			break
		}
		data = bytes.TrimSpace([]byte(s))
	}
	return data
}

func visualFromMap(m map[string]any) ImageVisualAnalysis {
	return ImageVisualAnalysis{
		ImageID:            pickString(m, "image_id", "imageId", "id"),
		FileName:           pickString(m, "file_name", "filename", "fileName", "name"),
		SubjectDescription: pickString(m, "subject_description", "subject", "description"),
		AtmosphereVibe:     pickString(m, "atmosphere_vibe", "atmosphere", "vibe"),
		VisualDetails:      StringList(pickStringSlice(m, "visual_details", "details", "visualDetails")),
		ImageQualityScore:  pickString(m, "image_quality_score", "quality", "score"),
		HighlightFeature:   pickString(m, "highlight_feature", "highlight", "anchor"),
	}
}

func editFromMap(m map[string]any) ImageEditPlan {
	return ImageEditPlan{
		ImageID:                    pickString(m, "image_id", "imageId", "id"),
		FileName:                   pickString(m, "file_name", "filename", "fileName", "name"),
		OverallEditStrategy:        pickString(m, "overall_edit_strategy", "edit_strategy", "strategy"),
		CropSuggestion:             pickString(m, "crop_suggestion", "crop"),
		LightColorAdjustment:       pickString(m, "light_color_adjustment", "light", "color"),
		FilterSuggestion:           pickString(m, "filter_suggestion", "filter"),
		TextOverlaySuggestion:      pickString(m, "text_overlay_suggestion", "text", "overlay"),
		BeautyAdjustmentSuggestion: pickString(m, "beauty_adjustment_suggestion", "beauty"),
		IsRecommendedAsCover:       pickBool(m, "is_recommended_as_cover", "cover", "is_cover"),
		RiskAndPitfallNotes:        pickString(m, "risk_and_pitfall_notes", "risk", "notes"),
	}
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(anyToString(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func pickStringSlice(m map[string]any, keys ...string) []string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			continue
		}
		out, err := decodeJSONArray(raw)
		if err == nil && len(out) > 0 {
			if len(out) == 1 && strings.Contains(out[0], ",") {
				parts := strings.Split(out[0], ",")
				clean := make([]string, 0, len(parts))
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						clean = append(clean, p)
					}
				}
				if len(clean) > 1 {
					return clean
				}
			}
			return out
		}
	}
	return nil
}

func pickBool(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case bool:
			return t
		case string:
			s := strings.ToLower(strings.TrimSpace(t))
			return s == "true" || s == "1" || s == "yes"
		case float64:
			return t != 0
		}
	}
	return false
}

func anyToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%v", t)
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		s := string(b)
		if len(s) >= 2 && s[0] == '"' {
			var out string
			if json.Unmarshal(b, &out) == nil {
				return out
			}
		}
		return s
	}
}
