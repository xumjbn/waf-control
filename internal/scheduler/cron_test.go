package scheduler

import (
	"testing"
	"time"
)

func TestParseInvalid(t *testing.T) {
	cases := []string{"", "* * * *", "60 * * * *", "* 24 * * *", "a b c d e", "*/0 * * * *"}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("Parse(%q) 应当报错，但通过了", c)
		}
	}
}

func TestMatchesEveryMinute(t *testing.T) {
	s, err := Parse("* * * * *")
	if err != nil {
		t.Fatal(err)
	}
	// 任意时刻都应命中。
	if !s.Matches(time.Date(2026, 5, 30, 13, 47, 0, 0, time.UTC)) {
		t.Error("* * * * * 应命中任意时刻")
	}
}

func TestMatchesDaily(t *testing.T) {
	s, err := Parse("0 0 * * *") // 每天 0:00
	if err != nil {
		t.Fatal(err)
	}
	midnight := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	if !s.Matches(midnight) {
		t.Error("0 0 * * * 应命中 00:00")
	}
	if s.Matches(midnight.Add(time.Minute)) {
		t.Error("0 0 * * * 不应命中 00:01")
	}
}

func TestMatchesStepAndRange(t *testing.T) {
	s, err := Parse("*/15 9-17 * * *") // 工作时间每 15 分钟
	if err != nil {
		t.Fatal(err)
	}
	hit := time.Date(2026, 5, 30, 14, 30, 0, 0, time.UTC)
	if !s.Matches(hit) {
		t.Error("*/15 9-17 应命中 14:30")
	}
	miss := time.Date(2026, 5, 30, 14, 31, 0, 0, time.UTC)
	if s.Matches(miss) {
		t.Error("*/15 应当不命中 14:31")
	}
	off := time.Date(2026, 5, 30, 8, 0, 0, 0, time.UTC)
	if s.Matches(off) {
		t.Error("9-17 不应命中 08:00")
	}
}

func TestMatchesDOW(t *testing.T) {
	s, err := Parse("0 12 * * 1") // 每周一 12:00（周一=1）
	if err != nil {
		t.Fatal(err)
	}
	// 2026-06-01 是周一。
	mon := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	if mon.Weekday() != time.Monday {
		t.Fatalf("测试前提错误：2026-06-01 不是周一而是 %v", mon.Weekday())
	}
	if !s.Matches(mon) {
		t.Error("应命中周一 12:00")
	}
	tue := mon.Add(24 * time.Hour)
	if s.Matches(tue) {
		t.Error("不应命中周二 12:00")
	}
}

func TestNext(t *testing.T) {
	s, err := Parse("0 0 * * *")
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	next := s.Next(from)
	want := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("Next = %v，期望 %v", next, want)
	}
}
