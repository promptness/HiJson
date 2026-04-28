package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetStartupFile checks if a file was passed as a command-line argument (e.g. drag onto icon)
func (a *App) GetStartupFile() *JsonResult {
	args := os.Args
	if len(args) > 1 {
		filePath := args[1]
		if _, err := os.Stat(filePath); err == nil {
			return a.ReadFilePath(filePath)
		}
	}
	return &JsonResult{Success: false, Error: "no startup file"}
}

// ReadFilePath reads a file by its path and returns formatted content
func (a *App) ReadFilePath(filePath string) *JsonResult {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &JsonResult{Success: false, Error: fmt.Sprintf("读取文件失败: %v", err)}
	}
	content := string(data)
	// Try to format
	if json.Valid([]byte(strings.TrimSpace(content))) {
		result := a.FormatJSON(content)
		if result.Success {
			return result
		}
	}
	return &JsonResult{Success: true, Text: content}
}

// FormatJSON formats JSON with indentation (preserves key order)
func (a *App) FormatJSON(input string) *JsonResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &JsonResult{Success: false, Error: "JSON文本为空"}
	}
	// Validate JSON
	if !json.Valid([]byte(input)) {
		return &JsonResult{Success: false, Error: "非法JSON字符串"}
	}
	// Use json.Indent to preserve key order
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(input), "", "  "); err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}
	return &JsonResult{Success: true, Text: buf.String()}
}

// FormatSortedJSON formats JSON with sorted keys
func (a *App) FormatSortedJSON(input string) *JsonResult {
	return processJSON(input, true, false, false)
}

// CompressJSON compresses JSON (removes whitespace)
func (a *App) CompressJSON(input string) *JsonResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &JsonResult{Success: false, Error: "JSON文本为空"}
	}
	if !json.Valid([]byte(input)) {
		return &JsonResult{Success: false, Error: "非法JSON字符串"}
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(input)); err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}
	return &JsonResult{Success: true, Text: buf.String()}
}

// FilterJSON removes null/empty values from JSON while preserving key order
func (a *App) FilterJSON(input string) *JsonResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &JsonResult{Success: false, Error: "JSON文本为空"}
	}
	if !json.Valid([]byte(input)) {
		return &JsonResult{Success: false, Error: "非法JSON字符串"}
	}

	filtered, err := filterOrdered(json.NewDecoder(strings.NewReader(input)))
	if err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}
	if filtered == nil {
		return &JsonResult{Success: true, Text: "null"}
	}
	out, err := json.Marshal(filtered)
	if err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, out, "", "  "); err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}
	return &JsonResult{Success: true, Text: buf.String()}
}

// orderedMap preserves insertion order of keys
type orderedMap struct {
	keys   []string
	values map[string]interface{}
}

func (o *orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	for _, k := range o.keys {
		v := o.values[k]
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
		key, _ := json.Marshal(k)
		buf.Write(key)
		buf.WriteByte(':')
		val, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// filterOrdered parses JSON preserving key order and removes empty values
func filterOrdered(dec *json.Decoder) (interface{}, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := t.(type) {
	case json.Delim:
		if v == '{' {
			om := &orderedMap{values: make(map[string]interface{})}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key := kt.(string)
				val, err := filterOrdered(dec)
				if err != nil {
					return nil, err
				}
				if !isEmpty(val) {
					om.keys = append(om.keys, key)
					om.values[key] = val
				}
			}
			// consume closing }
			dec.Token()
			if len(om.keys) == 0 {
				return nil, nil
			}
			return om, nil
		} else if v == '[' {
			var arr []interface{}
			for dec.More() {
				val, err := filterOrdered(dec)
				if err != nil {
					return nil, err
				}
				if !isEmpty(val) {
					arr = append(arr, val)
				}
			}
			// consume closing ]
			dec.Token()
			if len(arr) == 0 {
				return nil, nil
			}
			return arr, nil
		}
	case nil:
		return nil, nil
	case string:
		if v == "" || strings.EqualFold(v, "null") {
			return nil, nil
		}
		return v, nil
	default:
		return v, nil
	}
	return nil, nil
}

func isEmpty(v interface{}) bool {
	return v == nil
}

// DeepParseJSON expands nested JSON strings while preserving key order
func (a *App) DeepParseJSON(input string) *JsonResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &JsonResult{Success: false, Error: "JSON文本为空"}
	}
	if !json.Valid([]byte(input)) {
		return &JsonResult{Success: false, Error: "非法JSON字符串"}
	}
	// First format to get consistent output
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(input), "", "  "); err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}
	// Repeatedly expand embedded JSON strings until no more changes
	result := buf.String()
	for i := 0; i < 10; i++ {
		expanded := expandJSONStrings(result)
		if expanded == result {
			break
		}
		result = expanded
	}
	return &JsonResult{Success: true, Text: result}
}

// expandJSONStrings finds string values containing JSON and expands them in place
func expandJSONStrings(input string) string {
	var result strings.Builder
	i := 0
	runes := []byte(input)
	n := len(runes)

	for i < n {
		if runes[i] == '"' {
			// Find the full JSON string value
			strStart := i
			i++ // skip opening quote
			for i < n {
				if runes[i] == '\\' {
					i += 2
					continue
				}
				if runes[i] == '"' {
					i++ // skip closing quote
					break
				}
				i++
			}
			rawStr := string(runes[strStart:i])
			// Unquote the string
			var unquoted string
			if err := json.Unmarshal([]byte(rawStr), &unquoted); err == nil {
				trimmed := strings.TrimSpace(unquoted)
				if len(trimmed) >= 2 && ((trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}') || (trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']')) {
					if json.Valid([]byte(trimmed)) {
						// Determine indentation: look back to find the indent of current line
						indent := ""
						lineStart := strStart - 1
						for lineStart >= 0 && runes[lineStart] != '\n' {
							lineStart--
						}
						lineStart++
						// Find colon position to get the base indent
						for j := lineStart; j < strStart; j++ {
							if runes[j] == ' ' {
								indent += " "
							} else {
								break
							}
						}
						var buf bytes.Buffer
						if err := json.Indent(&buf, []byte(trimmed), indent, "  "); err == nil {
							result.WriteString(buf.String())
							continue
						}
					}
				}
			}
			result.WriteString(rawStr)
		} else {
			result.WriteByte(runes[i])
			i++
		}
	}
	return result.String()
}

// RemoveNewlines removes \n from text
func (a *App) RemoveNewlines(input string) string {
	return strings.ReplaceAll(input, "\n", "")
}

// RemoveBackslashes removes \ from text
func (a *App) RemoveBackslashes(input string) string {
	return strings.ReplaceAll(input, "\\", "")
}

// UnescapeString unescapes Java/JSON escape sequences
func (a *App) UnescapeString(input string) string {
	// Try JSON unquote
	var s string
	wrapped := `"` + input + `"`
	if err := json.Unmarshal([]byte(wrapped), &s); err == nil {
		return s
	}
	return input
}

// OpenFile opens a file dialog and reads the file
func (a *App) OpenFile() *JsonResult {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "打开JSON文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json;*.txt"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil || file == "" {
		return &JsonResult{Success: false, Error: "未选择文件"}
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return &JsonResult{Success: false, Error: fmt.Sprintf("读取文件失败: %v", err)}
	}

	content := string(data)
	// Auto format
	result := processJSON(content, false, false, false)
	if !result.Success {
		// Return raw content even if not valid JSON
		return &JsonResult{Success: true, Text: content}
	}
	return result
}

// SaveFile saves content to a file
func (a *App) SaveFile(content string) *JsonResult {
	file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "保存JSON文件",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files", Pattern: "*.json"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil || file == "" {
		return &JsonResult{Success: false, Error: "未选择文件"}
	}

	err = os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		return &JsonResult{Success: false, Error: fmt.Sprintf("保存文件失败: %v", err)}
	}
	return &JsonResult{Success: true, Text: "保存成功"}
}

// JsonResult represents the result of a JSON operation
type JsonResult struct {
	Success bool   `json:"success"`
	Text    string `json:"text"`
	Error   string `json:"error"`
}

func processJSON(input string, sorted bool, compress bool, _ bool) *JsonResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return &JsonResult{Success: false, Error: "JSON文本为空"}
	}

	var raw interface{}
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}

	if sorted {
		raw = sortKeys(raw)
	}

	var out []byte
	var err error
	if compress {
		out, err = json.Marshal(raw)
	} else {
		out, err = json.MarshalIndent(raw, "", "  ")
	}
	if err != nil {
		return &JsonResult{Success: false, Error: err.Error()}
	}

	return &JsonResult{Success: true, Text: string(out)}
}

// sortKeys recursively sorts map keys
func sortKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		sorted := make(map[string]interface{})
		for k, item := range val {
			sorted[k] = sortKeys(item)
		}
		return sorted
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = sortKeys(item)
		}
		return result
	default:
		return v
	}
}

// filterValue recursively removes null, empty string, empty map, empty array
func filterValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		if len(val) == 0 {
			return nil
		}
		result := make(map[string]interface{})
		for k, item := range val {
			filtered := filterValue(item)
			if filtered != nil {
				result[k] = filtered
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case []interface{}:
		if len(val) == 0 {
			return nil
		}
		var result []interface{}
		for _, item := range val {
			filtered := filterValue(item)
			if filtered != nil {
				result = append(result, filtered)
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result
	case string:
		if val == "" || strings.EqualFold(val, "null") {
			return nil
		}
		return val
	default:
		return val
	}
}

// deepExpand recursively expands JSON strings within values
func deepExpand(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, item := range val {
			result[k] = deepExpand(item)
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(val))
		for _, item := range val {
			result = append(result, deepExpand(item))
		}
		return result
	case string:
		trimmed := strings.TrimSpace(val)
		if len(trimmed) >= 2 && ((trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}') || (trimmed[0] == '[' && trimmed[len(trimmed)-1] == ']')) {
			var parsed interface{}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return deepExpand(parsed)
			}
		}
		return val
	default:
		return val
	}
}
