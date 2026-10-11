package parity

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/unraid/apprise-go/internal/notify"
	"github.com/unraid/apprise-go/internal/testutil"
)

// TestProviderRegressionRequestParity compares complete Send sequences and
// delivery outcomes for the template, recipient, and optional-target fixes.
func TestProviderRegressionRequestParity(t *testing.T) {
	testutil.RequirePythonApprise(t)
	tests := []struct {
		name string
		url  string
	}{
		{"mailersend/copy-and-reply", "mailersend://apikey123:user@example.com?cc=bad,copy@example.com&bcc=bad,blind@example.com&reply=user@example.com"},
		{"mailersend/bracketed-copies", "mailersend://apikey123:user@example.com/one@example.com/two@example.com?cc=bad,%3Ccopy@example.com%3E,%3Cone@example.com%3E&bcc=bad,%3Cblind@example.com%3E&reply=Reply%20Name%20%3Cuser@example.com%3E"},
		{"postmark/bracketed-copies", "postmark://apikey123:sender@example.com/one@example.com/two@example.com?cc=bad,%3Cone@example.com%3E&bcc=bad,%3Cblind@example.com%3E"},
		{"postmark/blind-copy-exclusions", "postmark://apikey123:sender@example.com/one@example.com?cc=bad,%3Cblind@example.com%3E,%3Cone@example.com%3E&bcc=bad,%3Cone@example.com%3E"},
		{"kook/empty-required", "kook://bottoken123?optional=no"},
		{"kook/empty-optional", "kook://bottoken123?optional=yes"},
		{"kook/invalid-required", "kook://bottoken123/invalid-channel?optional=no"},
		{"kook/invalid-optional", "kook://bottoken123/invalid-channel?optional=yes"},
	}
	for _, provider := range []struct {
		name  string
		url   string
		valid string
	}{
		{"slack", "slack://T111/F222/X333/Y444?mode=workflow&template=", `{"blocks":[{"type":"section","text":{"type":"mrkdwn","text":"{{app_title}}: {{app_body}}"}}]}`},
		{"discord", "discord://1234567890/abcdefghijklmnopqrstuvwxyz?template=", `{"content":"{{app_title}}: {{app_body}}","embeds":[]}`},
		{"guilded", "guilded://1234567890/abcdefghijklmnopqrstuvwxyz?template=", `{"content":"{{app_title}}: {{app_body}}","embeds":[]}`},
	} {
		payloads := []string{provider.valid, `{}`, `{"attachments":[{}]}`, `{"content":42,"embeds":["bad"]}`, `{"blocks":[{}]}`}
		for i, payload := range payloads {
			path := filepath.Join(t.TempDir(), "template.json")
			if err := os.WriteFile(path, []byte(payload), 0600); err != nil {
				t.Fatal(err)
			}
			label := "invalid"
			if i == 0 {
				label = "valid"
			}
			tests = append(tests, struct {
				name string
				url  string
			}{provider.name + "/" + label + "/" + payload, provider.url + url.QueryEscape(path)})
		}
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pythonSpecs, pythonSuccess := testutil.CapturePythonRequestsWithTypeResult(t, tt.url, "body", "title", "", notify.NotifyInfo)
			parsed, err := notify.ParseURL(tt.url)
			if err != nil {
				t.Fatal(err)
			}
			goSpecs, err := testutil.CaptureGoRequestsResult(t, func() error {
				target, err := notify.NewTarget(parsed)
				if err != nil {
					return err
				}
				return notify.DispatchSendWithInput(target, parsed, "body", "title", "", notify.NotifyInfo, nil)
			})
			if pythonSuccess == nil {
				t.Fatal("Python capture omitted the delivery outcome")
			}
			assertNotifySuccessMatches(t, pythonSuccess, err)
			assertRequestSpecSequenceMatches(t, pythonSpecs, goSpecs)
		})
	}
}
