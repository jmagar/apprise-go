package notify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderTemplateValidation(t *testing.T) {
	for _, scheme := range []string{"slack", "discord", "guilded"} {
		for _, payload := range []string{`{}`, `{"attachments":[{}]}`, `{"content":42,"embeds":["bad"]}`, `{"blocks":[{}]}`} {
			t.Run(scheme+"/invalid/"+payload, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "template.json")
				if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
					t.Fatal(err)
				}
				url := scheme + "://token/token?template=" + path
				if scheme == "slack" {
					url = "slack://token/token/token?template=" + path
				}
				parsed, err := ParseURL(url)
				if err != nil {
					t.Fatal(err)
				}
				target, err := NewTarget(parsed)
				if err != nil {
					t.Fatal(err)
				}
				builder := target.(interface {
					BuildRequest(string, string, NotifyType) (RequestSpec, error)
				})
				if _, err := builder.BuildRequest("body", "title", NotifyInfo); err == nil {
					t.Fatal("invalid provider template was accepted")
				}
			})
		}
		t.Run(scheme+"/valid", func(t *testing.T) {
			payload := `{"content":"{{app_title}}: {{app_body}}"}`
			if scheme == "slack" {
				payload = `{"blocks":[{"type":"section","text":{"type":"mrkdwn","text":"{{app_title}}: {{app_body}}"}}]}`
			}
			path := filepath.Join(t.TempDir(), "template.json")
			if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
				t.Fatal(err)
			}
			url := scheme + "://token/token?template=" + path
			if scheme == "slack" {
				url = "slack://token/token/token?template=" + path
			}
			parsed, err := ParseURL(url)
			if err != nil {
				t.Fatal(err)
			}
			target, err := NewTarget(parsed)
			if err != nil {
				t.Fatal(err)
			}
			builder := target.(interface {
				BuildRequest(string, string, NotifyType) (RequestSpec, error)
			})
			spec, err := builder.BuildRequest("body", "title", NotifyInfo)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(spec.Body, "title: body") {
				t.Fatalf("template content missing: %s", spec.Body)
			}
		})
	}
}

func TestMailerSendCopyRecipientsAndReply(t *testing.T) {
	parsed, err := ParseURL("mailersend://token:user@example.com?cc=bad,copy@example.com&bcc=bad,blind@example.com&reply=user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	target, err := NewMailerSendTarget(parsed)
	if err != nil {
		t.Fatal(err)
	}
	spec, err := target.BuildRequest("body", "title", NotifyInfo)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(spec.Body), &payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(spec.Body, `"bad"`) {
		t.Fatal("invalid copy recipient was sent")
	}
	if payload["reply_to"].(map[string]any)["email"] != "user@example.com" {
		t.Fatal("reply-to missing")
	}
	for _, key := range []string{"cc", "bcc"} {
		if len(payload[key].([]any)) != 1 {
			t.Fatalf("valid %s recipient missing", key)
		}
	}
}

func TestKookMissingTargetsRespectOptionalDelivery(t *testing.T) {
	for _, destination := range []string{"", "/invalid-channel"} {
		for _, optional := range []bool{false, true} {
			flag := "no"
			if optional {
				flag = "yes"
			}
			url := "kook://bottoken123" + destination + "?optional=" + flag
			t.Run(url, func(t *testing.T) {
				parsed, err := ParseURL(url)
				if err != nil {
					t.Fatal(err)
				}
				target, err := NewKookTarget(parsed)
				if err != nil {
					t.Fatalf("missing destinations must fail delivery, not construction: %v", err)
				}
				if _, err := target.BuildRequest("body", "title", NotifyInfo); err == nil {
					t.Fatal("missing destinations produced a request")
				}
				if err := target.Send("body", "title", NotifyInfo); err == nil {
					t.Fatal("missing destinations reported successful delivery")
				}
				err = DispatchSendWithInput(target, parsed, "body", "title", "", NotifyInfo, nil)
				if optional && err != nil {
					t.Fatalf("optional delivery failure was propagated: %v", err)
				}
				if !optional && err == nil {
					t.Fatal("required delivery failure was ignored")
				}
			})
		}
	}
}
