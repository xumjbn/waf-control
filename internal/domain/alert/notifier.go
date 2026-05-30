package alert

// notifier.go —— 告警渠道真实投递。替代之前 TestChannel 只写一条 INFO 事件的占位实现。
//
// 支持的真实通道：
//   webhook            POST JSON {level,subject,message,timestamp} 到 target/config.url
//   dingtalk / wechat  机器人 webhook，POST {"msgtype":"text","text":{"content":...}} 到 target
//   email              net/smtp 发送，config 提供 smtp_host/smtp_port/username/password/from
//
// 暂未实现（返回明确错误，不假装成功）：pagerduty / sms。
//
// 所有出站请求 8s 超时；SMTP 走 STARTTLS（587）或隐式（465 时调用方自行处理，
// 这里用标准 smtp.SendMail，多数自建/云 SMTP 587+STARTTLS 适用）。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type channelConfig struct {
	URL      string `json:"url"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort string `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
}

func parseChannelConfig(raw json.RawMessage) channelConfig {
	var c channelConfig
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &c)
	}
	return c
}

// Send 把一条告警通过指定渠道真实投递出去。subject/body 是告警标题与正文。
func Send(ctx context.Context, ch *Channel, level, subject, body string) error {
	cfg := parseChannelConfig(ch.Config)
	switch ch.Kind {
	case ChannelKindWebhook:
		url := cfg.URL
		if url == "" {
			url = ch.Target
		}
		return postJSON(ctx, url, map[string]any{
			"level":     level,
			"subject":   subject,
			"message":   body,
			"channel":   ch.Name,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	case ChannelKindDingTalk, ChannelKindWeChat:
		url := cfg.URL
		if url == "" {
			url = ch.Target
		}
		// 钉钉 / 企业微信群机器人统一 text 消息体。
		return postJSON(ctx, url, map[string]any{
			"msgtype": "text",
			"text":    map[string]string{"content": subject + "\n" + body},
		})
	case ChannelKindEmail:
		return sendEmail(cfg, ch.Target, subject, body)
	case ChannelKindPagerDuty, ChannelKindSMS:
		return fmt.Errorf("%s 渠道投递暂未实现（需接入对应服务商 SDK）", ch.Kind)
	default:
		return fmt.Errorf("未知渠道类型：%s", ch.Kind)
	}
}

func postJSON(ctx context.Context, url string, payload any) error {
	if url == "" {
		return fmt.Errorf("渠道未配置投递地址（target / config.url 为空）")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("非法投递地址：%q", url)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化投递负载: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造请求: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("投递失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("投递返回非 2xx 状态：%d", resp.StatusCode)
	}
	return nil
}

func sendEmail(cfg channelConfig, to, subject, body string) error {
	if cfg.SMTPHost == "" {
		return fmt.Errorf("邮件渠道未配置 smtp_host")
	}
	if to == "" {
		return fmt.Errorf("邮件渠道未配置收件人（target 为空）")
	}
	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("非法 smtp_port：%q", port)
	}
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}
	addr := cfg.SMTPHost + ":" + port

	// 多收件人：target 支持逗号分隔。
	recipients := splitRecipients(to)
	msg := buildEmailMessage(from, recipients, subject, body)

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}
	if err := smtp.SendMail(addr, auth, from, recipients, msg); err != nil {
		return fmt.Errorf("SMTP 发送失败: %w", err)
	}
	return nil
}

func splitRecipients(to string) []string {
	parts := strings.Split(to, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func buildEmailMessage(from string, to []string, subject, body string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
