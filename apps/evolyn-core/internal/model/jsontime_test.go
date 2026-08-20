package model

import (
	"encoding/json"
	"testing"
	"time"
)

// 东八区基准时刻：2026-08-20 12:40:48 +08:00
var baseInstant = time.Date(2026, 8, 20, 12, 40, 48, 416932000, cstLocation)

func TestMarshalJSON(t *testing.T) {
	cases := []struct {
		name string
		in   JSONTime
		want string
	}{
		{"统一格式去掉微秒与偏移", JSONTime(baseInstant), `"2026-08-20 12:40:48"`},
		{"UTC 输入归一为东八区墙钟", JSONTime(baseInstant.In(time.UTC)), `"2026-08-20 12:40:48"`},
		{"零值输出空串", JSONTime{}, `""`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(c.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
		})
	}
}

func TestMarshalJSONNilPointerAsNull(t *testing.T) {
	var p *JSONTime
	got, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal nil pointer: %v", err)
	}
	if string(got) != "null" {
		t.Fatalf("got %s, want null", got)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{"统一格式按东八区解释", `"2026-08-20 12:40:48"`, baseInstant.Truncate(time.Second), false},
		{"RFC3339 带偏移", `"2026-08-20T12:40:48+08:00"`, baseInstant.Truncate(time.Second), false},
		{"RFC3339Nano 带微秒", `"2026-08-20T12:40:48.416932+08:00"`, baseInstant, false},
		{"空串归零值", `""`, time.Time{}, false},
		{"null 归零值", `null`, time.Time{}, false},
		{"非法输入报错", `"2026/08/20 12:40"`, time.Time{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got JSONTime
			err := json.Unmarshal([]byte(c.in), &got)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expect error, got %v", got.Time())
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !got.Time().Equal(c.want) {
				t.Fatalf("got %v, want %v", got.Time(), c.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	orig := JSONTime(baseInstant)
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back JSONTime
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// 统一格式为秒级，往返后亚秒精度按设计丢弃
	if !back.Time().Equal(orig.Time().Truncate(time.Second)) {
		t.Fatalf("round trip mismatch: %v != %v", back.Time(), orig.Time())
	}
}

func TestScan(t *testing.T) {
	t.Run("time.Time 来源(pgx/GORM 自动时间戳)", func(t *testing.T) {
		var got JSONTime
		if err := got.Scan(baseInstant); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if !got.Time().Equal(baseInstant) {
			t.Fatalf("got %v, want %v", got.Time(), baseInstant)
		}
	})
	t.Run("字符串来源(统一格式)", func(t *testing.T) {
		var got JSONTime
		if err := got.Scan("2026-08-20 12:40:48"); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if !got.Time().Equal(baseInstant.Truncate(time.Second)) {
			t.Fatalf("got %v", got.Time())
		}
	})
	t.Run("字符串来源(RFC3339)", func(t *testing.T) {
		var got JSONTime
		if err := got.Scan([]byte("2026-08-20T04:40:48Z")); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if !got.Time().Equal(baseInstant.Truncate(time.Second)) {
			t.Fatalf("got %v", got.Time())
		}
	})
	t.Run("NULL 归零值", func(t *testing.T) {
		var got JSONTime
		if err := got.Scan(nil); err != nil {
			t.Fatalf("scan nil: %v", err)
		}
		if !got.IsZero() {
			t.Fatalf("expect zero, got %v", got.Time())
		}
	})
	t.Run("不支持类型报错", func(t *testing.T) {
		var got JSONTime
		if err := got.Scan(123); err == nil {
			t.Fatal("expect error")
		}
	})
}

func TestValue(t *testing.T) {
	v, err := JSONTime(baseInstant).Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	tt, ok := v.(time.Time)
	if !ok || !tt.Equal(baseInstant) {
		t.Fatalf("value got %v", v)
	}
}

func TestBaseModelJSONShape(t *testing.T) {
	// 基类字段出网走统一格式，确认字段名与格式同时符合预期
	type payload struct {
		PlatformBaseModel
	}
	data, err := json.Marshal(payload{PlatformBaseModel{CreatedAt: JSONTime(baseInstant), UpdatedAt: JSONTime{}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"createdAt":"2026-08-20 12:40:48","updatedAt":""}`
	if string(data) != want {
		t.Fatalf("got %s, want %s", data, want)
	}
}
