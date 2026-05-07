package src

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// ── Typy ─────────────────────────────────────────────────────────────────────

type HkValueType int

const (
	HkString HkValueType = iota
	HkNumber
	HkBool
	HkArray
	HkMap
)

// HkValue przechowuje wartość .hk (union przez interface{}).
type HkValue struct {
	Type    HkValueType
	Str     string
	Num     float64
	Bool    bool
	Array   []HkValue
	Map     map[string]HkValue
	MapKeys []string // zachowaj kolejność kluczy
}

// HkConfig — mapa sekcji najwyższego poziomu.
type HkConfig struct {
	Sections     map[string]HkValue
	SectionOrder []string
}

// ── Konstruktory ──────────────────────────────────────────────────────────────

func hkStr(s string) HkValue  { return HkValue{Type: HkString, Str: s} }
func hkNum(n float64) HkValue { return HkValue{Type: HkNumber, Num: n} }
func hkBool(b bool) HkValue   { return HkValue{Type: HkBool, Bool: b} }
func hkArr(a []HkValue) HkValue { return HkValue{Type: HkArray, Array: a} }
func hkMap() HkValue { return HkValue{Type: HkMap, Map: make(map[string]HkValue)} }

// AsString konwertuje String/Number/Bool na string.
func (v HkValue) AsString() (string, error) {
	switch v.Type {
	case HkString:
		return v.Str, nil
	case HkNumber:
		if v.Num == math.Trunc(v.Num) {
			return strconv.FormatInt(int64(v.Num), 10), nil
		}
		return strconv.FormatFloat(v.Num, 'f', -1, 64), nil
	case HkBool:
		if v.Bool {
			return "true", nil
		}
		return "false", nil
	}
	return "", fmt.Errorf("hk: nie można przekonwertować %v na string", v.Type)
}

func (v HkValue) AsNumber() (float64, error) {
	if v.Type == HkNumber {
		return v.Num, nil
	}
	return 0, fmt.Errorf("hk: oczekiwano number, znaleziono %v", v.Type)
}

func (v HkValue) AsBool() (bool, error) {
	if v.Type == HkBool {
		return v.Bool, nil
	}
	return false, fmt.Errorf("hk: oczekiwano bool, znaleziono %v", v.Type)
}

func (v HkValue) AsInt() (int, error) {
	n, err := v.AsNumber()
	return int(n), err
}

// Get zwraca wartość klucza z mapy lub błąd.
func (v HkValue) Get(key string) (HkValue, error) {
	if v.Type != HkMap {
		return HkValue{}, fmt.Errorf("hk: nie jest mapą")
	}
	val, ok := v.Map[key]
	if !ok {
		return HkValue{}, fmt.Errorf("hk: brakujący klucz %q", key)
	}
	return val, nil
}

// ── Parser ────────────────────────────────────────────────────────────────────

// ParseHK parsuje string w formacie .hk i zwraca HkConfig.
func ParseHK(input string) (*HkConfig, error) {
	lines := strings.Split(input, "\n")
	cfg := &HkConfig{Sections: make(map[string]HkValue)}
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "!") {
			i++
			continue
		}
		if strings.HasPrefix(line, "[") {
			close := strings.Index(line, "]")
			if close < 0 {
				return nil, fmt.Errorf("hk: linia %d: niedomknięty nagłówek sekcji", i+1)
			}
			sname := strings.TrimSpace(line[1:close])
			if sname == "" {
				return nil, fmt.Errorf("hk: linia %d: pusta nazwa sekcji", i+1)
			}
			// Znajdź koniec sekcji
			end := i + 1
			for end < len(lines) {
				nl := strings.TrimSpace(lines[end])
				if strings.HasPrefix(nl, "[") {
					break
				}
				end++
			}
			sectionLines := lines[i+1 : end]
			m, err := parseMap(1, sectionLines, i+1)
			if err != nil {
				return nil, err
			}
			cfg.SectionOrder = append(cfg.SectionOrder, sname)
			cfg.Sections[sname] = m
			i = end
		} else {
			return nil, fmt.Errorf("hk: linia %d: oczekiwano nagłówka sekcji [nazwa]", i+1)
		}
	}
	return cfg, nil
}

// parseMap parsuje mapę na danym poziomie zagnieżdżenia.
// level=1 → "->" , level=2 → "-->" itd.
func parseMap(level int, lines []string, startLine int) (HkValue, error) {
	result := hkMap()
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "!") {
			i++
			continue
		}
		// Policz myślniki
		dashes := 0
		for _, c := range trimmed {
			if c == '-' {
				dashes++
			} else {
				break
			}
		}
		if dashes == 0 {
			return HkValue{}, fmt.Errorf("hk: linia %d: oczekiwano myślnika", startLine+i+1)
		}
		if dashes != level {
			break // inny poziom — oddaj kontrolę
		}
		// Po myślnikach: ">"
		afterDashes := strings.TrimSpace(trimmed[dashes:])
		if !strings.HasPrefix(afterDashes, ">") {
			return HkValue{}, fmt.Errorf("hk: linia %d: oczekiwano '>' po myślnikach", startLine+i+1)
		}
		afterGT := strings.TrimSpace(afterDashes[1:])
		if afterGT == "" {
			return HkValue{}, fmt.Errorf("hk: linia %d: brak klucza po '>'", startLine+i+1)
		}

		// Klucz-wartość (zawiera "=>")?
		arrowIdx := strings.Index(afterGT, "=>")
		if arrowIdx >= 0 {
			key := strings.TrimSpace(afterGT[:arrowIdx])
			valStr := strings.TrimSpace(afterGT[arrowIdx+2:])
			key = unquoteKey(key)
			val, err := parseValue(valStr, startLine+i+1)
			if err != nil {
				return HkValue{}, err
			}
			if err := insertKey(&result, key, val); err != nil {
				return HkValue{}, err
			}
			i++
		} else {
			// Mapa: klucz bez "=>"
			key := unquoteKey(strings.TrimSpace(afterGT))
			nextLevel := level + 1
			// Znajdź zasięg pod-linii
			j := i + 1
			for j < len(lines) {
				sl := strings.TrimSpace(lines[j])
				if sl == "" || strings.HasPrefix(sl, "!") {
					j++
					continue
				}
				sd := 0
				for _, c := range sl {
					if c == '-' {
						sd++
					} else {
						break
					}
				}
				if sd < nextLevel {
					break
				}
				j++
			}
			subLines := lines[i+1 : j]
			sub, err := parseMap(nextLevel, subLines, startLine+i+1)
			if err != nil {
				return HkValue{}, err
			}
			if err := insertKey(&result, key, sub); err != nil {
				return HkValue{}, err
			}
			i = j
		}
	}
	return result, nil
}

// insertKey wstawia klucz (obsługuje notację kropkową dla zagnieżdżania).
func insertKey(m *HkValue, key string, val HkValue) error {
	if strings.Contains(key, ".") && !strings.HasPrefix(key, ".") && !strings.HasSuffix(key, ".") {
		parts := strings.Split(key, ".")
		return insertNested(m, parts, val)
	}
	if _, exists := m.Map[key]; exists {
		return fmt.Errorf("hk: konflikt klucza %q", key)
	}
	m.Map[key] = val
	m.MapKeys = append(m.MapKeys, key)
	return nil
}

func insertNested(m *HkValue, keys []string, val HkValue) error {
	if len(keys) == 1 {
		m.Map[keys[0]] = val
		m.MapKeys = append(m.MapKeys, keys[0])
		return nil
	}
	sub, ok := m.Map[keys[0]]
	if !ok {
		sub = hkMap()
	}
	if sub.Type != HkMap {
		return fmt.Errorf("hk: konflikt klucza %q", keys[0])
	}
	if err := insertNested(&sub, keys[1:], val); err != nil {
		return err
	}
	m.Map[keys[0]] = sub
	if !ok {
		m.MapKeys = append(m.MapKeys, keys[0])
	}
	return nil
}

func unquoteKey(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		return strings.ReplaceAll(s[1:len(s)-1], `\"`, `"`)
	}
	return s
}

// parseValue parsuje wartość: bool, number, array, quoted string, plain string.
func parseValue(s string, lineNum int) (HkValue, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return HkValue{}, fmt.Errorf("hk: linia %d: pusta wartość", lineNum)
	}
	// Array
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return parseArray(s[1:len(s)-1], lineNum)
	}
	return parseSimpleValue(s, lineNum)
}

func parseArray(inner string, lineNum int) (HkValue, error) {
	var items []HkValue
	var cur strings.Builder
	inQ := false
	esc := false
	for _, c := range inner {
		if esc {
			cur.WriteRune(c)
			esc = false
			continue
		}
		switch c {
		case '\\':
			esc = true
		case '"':
			inQ = !inQ
			cur.WriteRune(c)
		case ',':
			if !inQ {
				t := strings.TrimSpace(cur.String())
				if t != "" {
					v, err := parseSimpleValue(t, lineNum)
					if err != nil {
						return HkValue{}, err
					}
					items = append(items, v)
				}
				cur.Reset()
			} else {
				cur.WriteRune(c)
			}
		default:
			cur.WriteRune(c)
		}
	}
	if t := strings.TrimSpace(cur.String()); t != "" {
		v, err := parseSimpleValue(t, lineNum)
		if err != nil {
			return HkValue{}, err
		}
		items = append(items, v)
	}
	return hkArr(items), nil
}

func parseSimpleValue(s string, _ int) (HkValue, error) {
	s = strings.TrimSpace(s)
	// Bool
	switch strings.ToLower(s) {
	case "true":
		return hkBool(true), nil
	case "false":
		return hkBool(false), nil
	}
	// Number
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return hkNum(n), nil
	}
	// Quoted string
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		inner := s[1 : len(s)-1]
		var b strings.Builder
		esc := false
		for _, c := range inner {
			if esc {
				switch c {
				case 'n':
					b.WriteByte('\n')
				case 't':
					b.WriteByte('\t')
				case 'r':
					b.WriteByte('\r')
				case '"':
					b.WriteByte('"')
				case '\\':
					b.WriteByte('\\')
				default:
					b.WriteRune(c)
				}
				esc = false
			} else if c == '\\' {
				esc = true
			} else {
				b.WriteRune(c)
			}
		}
		return hkStr(b.String()), nil
	}
	// Plain string
	return hkStr(s), nil
}

// ── Serializacja ──────────────────────────────────────────────────────────────

// SerializeHK serializuje HkConfig z powrotem do formatu .hk.
func SerializeHK(cfg *HkConfig) string {
	var sb strings.Builder
	for _, sname := range cfg.SectionOrder {
		val := cfg.Sections[sname]
		sb.WriteString(fmt.Sprintf("[%s]\n", sname))
		if val.Type == HkMap {
			serializeMap(val, 1, &sb)
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func serializeMap(m HkValue, level int, sb *strings.Builder) {
	prefix := strings.Repeat("-", level) + " > "
	keys := m.MapKeys
	if len(keys) == 0 {
		for k := range m.Map {
			keys = append(keys, k)
		}
	}
	for _, key := range keys {
		val := m.Map[key]
		if val.Type == HkMap {
			sb.WriteString(fmt.Sprintf("%s%s\n", prefix, key))
			serializeMap(val, level+1, sb)
		} else {
			sb.WriteString(fmt.Sprintf("%s%s => %s\n", prefix, key, serializeValue(val)))
		}
	}
}

func serializeValue(v HkValue) string {
	switch v.Type {
	case HkString:
		s := v.Str
		if strings.ContainsAny(s, `, "]\n`) {
			return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
		}
		return s
	case HkNumber:
		if v.Num == math.Trunc(v.Num) {
			return strconv.FormatInt(int64(v.Num), 10)
		}
		return strconv.FormatFloat(v.Num, 'f', -1, 64)
	case HkBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case HkArray:
		parts := make([]string, len(v.Array))
		for i, item := range v.Array {
			parts[i] = serializeValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case HkMap:
		return "<map>"
	}
	return ""
}

// ── Wczytywanie / zapis pliku ──────────────────────────────────────────────────

func LoadHKFile(path string) (*HkConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("nie można wczytać %s: %w", path, err)
	}
	cfg, err := ParseHK(string(data))
	if err != nil {
		return nil, fmt.Errorf("błąd parsowania %s: %w", path, err)
	}
	return cfg, nil
}

func WriteHKFile(path string, cfg *HkConfig) error {
	content := SerializeHK(cfg)
	return os.WriteFile(path, []byte(content+"\n"), 0644)
}
