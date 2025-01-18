//go:build ignore

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"
)

const tmpl = `// Code generated by go generate; DO NOT EDIT.
package session

import (
	"time"
	
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
)

var (
	{{ range .UserSettings }}
	// {{ .Doc }}
	User{{ replace .Name "." "" }} = NewUserSettingKey[{{ .Type }}]("{{ .Name }}")
	{{- end }}

	{{ range .BotSettings }}
	// {{ .Doc }}
	Bot{{ replace .Name "." "" }} = NewBotSettingKey[{{ .Type }}]("{{ .Name }}")
	{{- end }}
)
`

type SettingDef struct {
	Name string
	Type string
	Doc  string
}

func main() {
	userSettings := []SettingDef{
		// Basic settings
		{Name: "StreamerMode", Type: "bool", Doc: "StreamerMode controls streamer-friendly display"},
		{Name: "UserDefaultSort", Type: "enum.ReviewSortBy", Doc: "UserDefaultSort sets default user review sorting"},
		{Name: "GroupDefaultSort", Type: "enum.ReviewSortBy", Doc: "GroupDefaultSort sets default group review sorting"},
		{Name: "AppealDefaultSort", Type: "enum.AppealSortBy", Doc: "AppealDefaultSort sets default appeal sorting"},
		{Name: "AppealStatusFilter", Type: "enum.AppealStatus", Doc: "AppealStatusFilter sets appeal status filtering"},
		{Name: "ChatModel", Type: "enum.ChatModel", Doc: "ChatModel sets the AI chat model"},
		{Name: "ReviewMode", Type: "enum.ReviewMode", Doc: "ReviewMode sets the review mode"},
		{Name: "ReviewTargetMode", Type: "enum.ReviewTargetMode", Doc: "ReviewTargetMode sets the review target mode"},
		{Name: "LeaderboardPeriod", Type: "enum.LeaderboardPeriod", Doc: "LeaderboardPeriod sets the leaderboard time period"},

		// Chat message usage settings
		{Name: "ChatMessageUsage.FirstMessageTime", Type: "time.Time", Doc: "ChatMessageUsageFirstMessageTime tracks first message time in 24h period"},
		{Name: "ChatMessageUsage.MessageCount", Type: "int", Doc: "ChatMessageUsageMessageCount tracks message count in 24h period"},

		// CAPTCHA usage settings
		{Name: "CaptchaUsage.ReviewCount", Type: "int", Doc: "CaptchaUsageReviewCount tracks reviews since last CAPTCHA"},
	}

	botSettings := []SettingDef{
		// Basic settings
		{Name: "ReviewerIDs", Type: "[]uint64", Doc: "ReviewerIDs stores authorized reviewer IDs"},
		{Name: "AdminIDs", Type: "[]uint64", Doc: "AdminIDs stores authorized admin IDs"},
		{Name: "SessionLimit", Type: "uint64", Doc: "SessionLimit sets maximum concurrent sessions"},
		{Name: "WelcomeMessage", Type: "string", Doc: "WelcomeMessage sets the welcome message"},

		// Announcement settings
		{Name: "Announcement.Type", Type: "enum.AnnouncementType", Doc: "AnnouncementType sets the announcement type"},
		{Name: "Announcement.Message", Type: "string", Doc: "AnnouncementMessage sets the announcement message"},

		// API settings
		{Name: "APIKeys", Type: "[]types.APIKeyInfo", Doc: "APIKeys stores API key information"},
	}

	// Create template
	t := template.Must(template.New("settings").Funcs(template.FuncMap{
		"replace": strings.ReplaceAll,
	}).Parse(tmpl))

	data := struct {
		UserSettings []SettingDef
		BotSettings  []SettingDef
	}{
		UserSettings: userSettings,
		BotSettings:  botSettings,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Fatal(err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	// Write to settings_generated.go
	err = os.WriteFile("settings_generated.go", formatted, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
