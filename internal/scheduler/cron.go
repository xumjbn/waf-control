package scheduler

// cron.go —— 极小标准 5 字段 cron 解析与匹配，不依赖第三方库。
//
// 字段：分(0-59) 时(0-23) 日(1-31) 月(1-12) 周(0-6，0=周日)。
// 支持：* / N / N-M / */S / N-M/S / 逗号列表 A,B,C。
// 周/日同时受限时按标准 cron 取「或」匹配（与 Vixie cron 一致）。
//
// 用途：调度器每分钟 tick 时用 Matches(now) 判断定时任务是否到点。

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Schedule struct {
	minute  map[int]bool
	hour    map[int]bool
	dom     map[int]bool
	month   map[int]bool
	dow     map[int]bool
	domStar bool // 日字段是否为 *（决定 dom/dow 的或逻辑）
	dowStar bool // 周字段是否为 *
}

// Parse 解析标准 5 字段 cron 表达式。
func Parse(expr string) (*Schedule, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron 需 5 个字段，得到 %d：%q", len(fields), expr)
	}
	s := &Schedule{
		domStar: fields[2] == "*",
		dowStar: fields[4] == "*",
	}
	var err error
	if s.minute, err = parseField(fields[0], 0, 59); err != nil {
		return nil, fmt.Errorf("分字段: %w", err)
	}
	if s.hour, err = parseField(fields[1], 0, 23); err != nil {
		return nil, fmt.Errorf("时字段: %w", err)
	}
	if s.dom, err = parseField(fields[2], 1, 31); err != nil {
		return nil, fmt.Errorf("日字段: %w", err)
	}
	if s.month, err = parseField(fields[3], 1, 12); err != nil {
		return nil, fmt.Errorf("月字段: %w", err)
	}
	if s.dow, err = parseField(fields[4], 0, 6); err != nil {
		return nil, fmt.Errorf("周字段: %w", err)
	}
	return s, nil
}

// Matches 判断时刻 t（截到分钟）是否命中该 cron。
func (s *Schedule) Matches(t time.Time) bool {
	if !s.minute[t.Minute()] || !s.hour[t.Hour()] || !s.month[int(t.Month())] {
		return false
	}
	domMatch := s.dom[t.Day()]
	dowMatch := s.dow[int(t.Weekday())] // time.Weekday: 0=Sunday
	// 标准 cron：日/周都受限 → 或；任一为 * → 与（即另一个说了算）。
	if s.domStar && s.dowStar {
		return true
	}
	if s.domStar {
		return dowMatch
	}
	if s.dowStar {
		return domMatch
	}
	return domMatch || dowMatch
}

// Next 返回 t 之后下一次命中的时刻（用于回写 next_run_at）。最多向前找 366 天，
// 找不到返回零值。
func (s *Schedule) Next(t time.Time) time.Time {
	// 从下一分钟开始逐分钟试探。粗暴但够用（调度仅在启动 / 触发时算一次）。
	cur := t.Truncate(time.Minute).Add(time.Minute)
	limit := cur.Add(366 * 24 * time.Hour)
	for cur.Before(limit) {
		if s.Matches(cur) {
			return cur
		}
		cur = cur.Add(time.Minute)
	}
	return time.Time{}
}

func parseField(field string, min, max int) (map[int]bool, error) {
	out := map[int]bool{}
	for _, part := range strings.Split(field, ",") {
		step := 1
		rangePart := part
		if slash := strings.Index(part, "/"); slash >= 0 {
			stepStr := part[slash+1:]
			rangePart = part[:slash]
			n, err := strconv.Atoi(stepStr)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("非法步长 %q", part)
			}
			step = n
		}

		lo, hi := min, max
		if rangePart != "*" {
			if dash := strings.Index(rangePart, "-"); dash >= 0 {
				a, err1 := strconv.Atoi(rangePart[:dash])
				b, err2 := strconv.Atoi(rangePart[dash+1:])
				if err1 != nil || err2 != nil {
					return nil, fmt.Errorf("非法区间 %q", part)
				}
				lo, hi = a, b
			} else {
				n, err := strconv.Atoi(rangePart)
				if err != nil {
					return nil, fmt.Errorf("非法值 %q", part)
				}
				lo, hi = n, n
			}
		}
		if lo < min || hi > max || lo > hi {
			return nil, fmt.Errorf("越界 %q（允许 %d-%d）", part, min, max)
		}
		for v := lo; v <= hi; v += step {
			out[v] = true
		}
	}
	return out, nil
}
