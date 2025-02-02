//go:build ignore

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"text/template"
)

type KeyDef struct {
	Name string
	Type string
	Doc  string
}

const tmpl = `// Code generated by go generate; DO NOT EDIT.
package session

import (
    "time"
    
    "github.com/robalyx/rotector/internal/common/storage/database/types"
    "github.com/robalyx/rotector/internal/common/storage/database/types/enum"
    "github.com/robalyx/rotector/internal/common/client/ai"
	"github.com/robalyx/rotector/internal/common/queue"
	"github.com/robalyx/rotector/internal/worker/core"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/disgoorg/snowflake/v2"
)

var (
    {{ range .Keys }}
    // {{ .Doc }}
    {{ .Name }} = NewKey[{{ .Type }}]("{{ .Name }}")
    {{- end }}

    {{ range .BufferKeys }}
    // {{ .Doc }}
    {{ .Name }} = NewBufferKey("{{ .Name }}")
    {{- end }}
)
`

func main() {
	keys := []KeyDef{
		// Navigation related keys
		{Name: "MessageID", Type: "string", Doc: "MessageID stores the ID of the current message"},
		{Name: "CurrentPage", Type: "string", Doc: "CurrentPage stores the current page identifier"},
		{Name: "PreviousPages", Type: "[]string", Doc: "PreviousPages stores the navigation history"},

		// Pagination related keys
		{Name: "PaginationPage", Type: "int", Doc: "PaginationPage stores the current pagination page number"},
		{Name: "PaginationOffset", Type: "int", Doc: "PaginationOffset stores the starting offset"},
		{Name: "PaginationTotalItems", Type: "int", Doc: "PaginationTotalItems stores the total number of items"},
		{Name: "PaginationTotalPages", Type: "int", Doc: "PaginationTotalPages stores the total number of pages"},
		{Name: "PaginationHasNextPage", Type: "bool", Doc: "PaginationHasNextPage indicates if there is a next page"},
		{Name: "PaginationHasPrevPage", Type: "bool", Doc: "PaginationHasPrevPage indicates if there is a previous page"},
		{Name: "PaginationIsStreaming", Type: "bool", Doc: "PaginationIsStreaming indicates if image streaming is active"},

		// Statistics related keys
		{Name: "StatsIsRefreshed", Type: "bool", Doc: "StatsIsRefreshed indicates if the data has been refreshed"},
		{Name: "StatsUserCounts", Type: "*types.UserCounts", Doc: "StatsUserCounts stores user statistics"},
		{Name: "StatsGroupCounts", Type: "*types.GroupCounts", Doc: "StatsGroupCounts stores group statistics"},
		{Name: "StatsActiveUsers", Type: "[]snowflake.ID", Doc: "StatsActiveUsers stores the list of active reviewers"},
		{Name: "StatsVotes", Type: "*types.VoteAccuracy", Doc: "StatsVotes stores a user's voting statistics"},

		// Status related keys
		{Name: "StatusWorkers", Type: "[]core.Status", Doc: "StatusWorkers stores worker status information"},

		// Settings related keys
		{Name: "SettingName", Type: "string", Doc: "SettingName stores the name of the current setting"},
		{Name: "SettingType", Type: "string", Doc: "SettingType stores the type of the current setting"},
		{Name: "SettingValue", Type: "*Setting", Doc: "SettingValue stores the setting value"},
		{Name: "SettingDisplay", Type: "string", Doc: "SettingDisplay stores the display value of the setting"},
		{Name: "SettingCustomID", Type: "string", Doc: "SettingCustomID stores the custom identifier"},

		// User related keys
		{Name: "UserTarget", Type: "*types.ReviewUser", Doc: "UserTarget stores the currently selected user"},
		{Name: "UserFriends", Type: "[]*apiTypes.ExtendedFriend", Doc: "UserFriends stores the user's friend list"},
		{Name: "UserPresences", Type: "map[uint64]*apiTypes.UserPresenceResponse", Doc: "UserPresences stores friend presence information"},
		{Name: "UserFlaggedFriends", Type: "map[uint64]*types.ReviewUser", Doc: "UserFlaggedFriends stores flagged friends"},
		{Name: "UserGroups", Type: "[]*apiTypes.UserGroupRoles", Doc: "UserGroups stores the list of groups"},
		{Name: "UserFlaggedGroups", Type: "map[uint64]*types.ReviewGroup", Doc: "UserFlaggedGroups stores flagged groups"},
		{Name: "UserOutfits", Type: "[]*apiTypes.Outfit", Doc: "UserOutfits stores user outfits"},

		// Group related keys
		{Name: "GroupTarget", Type: "*types.ReviewGroup", Doc: "GroupTarget stores the currently selected group"},
		{Name: "GroupMemberIDs", Type: "[]uint64", Doc: "GroupMemberIDs stores member IDs for the current group"},
		{Name: "GroupMembers", Type: "map[uint64]*types.ReviewUser", Doc: "GroupMembers stores member details for the current group"},
		{Name: "GroupPageMembers", Type: "[]uint64", Doc: "GroupPageMembers stores the current page of group members"},
		{Name: "GroupInfo", Type: "*apiTypes.GroupResponse", Doc: "GroupInfo stores additional group information"},

		// Chat related keys
		{Name: "ChatHistory", Type: "ai.ChatHistory", Doc: "ChatHistory stores the conversation history"},
		{Name: "ChatContext", Type: "string", Doc: "ChatContext stores chat context information"},

		// Log related keys
		{Name: "LogActivities", Type: "[]*types.ActivityLog", Doc: "LogActivities stores activity logs"},
		{Name: "LogCursor", Type: "*types.LogCursor", Doc: "LogCursor stores the current log cursor"},
		{Name: "LogNextCursor", Type: "*types.LogCursor", Doc: "LogNextCursor stores the next log cursor"},
		{Name: "LogPrevCursors", Type: "[]*types.LogCursor", Doc: "LogPrevCursors stores previous log cursors"},
		{Name: "LogFilterDiscordID", Type: "uint64", Doc: "LogFilterDiscordID stores Discord ID filter"},
		{Name: "LogFilterUserID", Type: "uint64", Doc: "LogFilterUserID stores user ID filter"},
		{Name: "LogFilterGroupID", Type: "uint64", Doc: "LogFilterGroupID stores group ID filter"},
		{Name: "LogFilterReviewerID", Type: "uint64", Doc: "LogFilterReviewerID stores reviewer ID filter"},
		{Name: "LogFilterActivityType", Type: "enum.ActivityType", Doc: "LogFilterActivityType stores activity type filter"},
		{Name: "LogFilterActivityCategory", Type: "string", Doc: "LogFilterActivityCategory stores the currently selected activity category"},
		{Name: "LogFilterDateRangeStart", Type: "time.Time", Doc: "LogFilterDateRangeStart stores start date filter"},
		{Name: "LogFilterDateRangeEnd", Type: "time.Time", Doc: "LogFilterDateRangeEnd stores end date filter"},

		// Queue related keys
		{Name: "QueueUser", Type: "uint64", Doc: "QueueUser stores the queued user"},
		{Name: "QueueStatus", Type: "queue.Status", Doc: "QueueStatus stores the queue status"},
		{Name: "QueuePriority", Type: "queue.Priority", Doc: "QueuePriority stores the queue priority"},
		{Name: "QueuePosition", Type: "int", Doc: "QueuePosition stores the queue position"},
		{Name: "QueueHighCount", Type: "int", Doc: "QueueHighCount stores high priority queue count"},
		{Name: "QueueNormalCount", Type: "int", Doc: "QueueNormalCount stores normal priority queue count"},
		{Name: "QueueLowCount", Type: "int", Doc: "QueueLowCount stores low priority queue count"},

		// Appeal related keys
		{Name: "AppealList", Type: "[]*types.Appeal", Doc: "AppealList stores the current page of appeals"},
		{Name: "AppealSelected", Type: "*types.Appeal", Doc: "AppealSelected stores the currently selected appeal"},
		{Name: "AppealMessages", Type: "[]*types.AppealMessage", Doc: "AppealMessages stores messages for the current appeal"},
		{Name: "AppealCursor", Type: "*types.AppealTimeline", Doc: "AppealCursor stores the current cursor position"},
		{Name: "AppealNextCursor", Type: "*types.AppealTimeline", Doc: "AppealNextCursor stores the next cursor position"},
		{Name: "AppealPrevCursors", Type: "[]*types.AppealTimeline", Doc: "AppealPrevCursors stores previous cursor positions"},

		// Verify related keys
		{Name: "VerifyUserID", Type: "uint64", Doc: "VerifyUserID stores the user ID being verified"},
		{Name: "VerifyReason", Type: "string", Doc: "VerifyReason stores the verification reason"},
		{Name: "VerifyCode", Type: "string", Doc: "VerifyCode stores the verification code"},

		// CAPTCHA related keys
		{Name: "CaptchaAnswer", Type: "string", Doc: "CaptchaAnswer stores the CAPTCHA answer"},

		// Admin related keys
		{Name: "AdminAction", Type: "string", Doc: "AdminAction stores the current admin action"},
		{Name: "AdminActionID", Type: "string", Doc: "AdminActionID stores the admin action ID"},
		{Name: "AdminReason", Type: "string", Doc: "AdminReason stores the admin action reason"},
		{Name: "AdminBanReason", Type: "enum.BanReason", Doc: "AdminBanReason stores the ban reason"},
		{Name: "AdminBanExpiry", Type: "*time.Time", Doc: "AdminBanExpiry stores the ban expiry time"},
		{Name: "AdminBanInfo", Type: "*types.DiscordBan", Doc: "AdminBanInfo stores ban information"},

		// Leaderboard related keys
		{Name: "LeaderboardStats", Type: "[]*types.VoteAccuracy", Doc: "LeaderboardStats stores leaderboard statistics"},
		{Name: "LeaderboardUsernames", Type: "map[uint64]string", Doc: "LeaderboardUsernames stores usernames for the leaderboard"},
		{Name: "LeaderboardCursor", Type: "*types.LeaderboardCursor", Doc: "LeaderboardCursor stores the current leaderboard cursor"},
		{Name: "LeaderboardNextCursor", Type: "*types.LeaderboardCursor", Doc: "LeaderboardNextCursor stores the next leaderboard cursor"},
		{Name: "LeaderboardPrevCursors", Type: "[]*types.LeaderboardCursor", Doc: "LeaderboardPrevCursors stores previous leaderboard cursors"},
		{Name: "LeaderboardLastRefresh", Type: "time.Time", Doc: "LeaderboardLastRefresh stores the last refresh time"},
		{Name: "LeaderboardNextRefresh", Type: "time.Time", Doc: "LeaderboardNextRefresh stores the next refresh time"},
	}

	bufferKeys := []KeyDef{
		{Name: "ImageBuffer", Doc: "ImageBuffer stores binary image data"},
	}

	data := struct {
		Keys       []KeyDef
		BufferKeys []KeyDef
	}{
		Keys:       keys,
		BufferKeys: bufferKeys,
	}

	t := template.Must(template.New("keys").Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		log.Fatal(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile("keys_generated.go", formatted, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
