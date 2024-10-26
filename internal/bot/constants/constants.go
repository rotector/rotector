package constants

const (
	// Commands.
	DashboardCommandName = "dashboard"

	// Common.
	NotApplicable            = "N/A"
	ActionSelectMenuCustomID = "action"
	DefaultEmbedColor        = 0x312D2B
	StreamerModeEmbedColor   = 0x3E3769

	// Dashboard Menu.
	StartReviewCustomID   = "start_review"
	UserSettingsCustomID  = "user_settings"
	GuildSettingsCustomID = "guild_settings"

	// Review Menu.
	SortOrderSelectMenuCustomID = "sort_order"

	BanWithReasonModalCustomID = "ban_with_reason_modal"

	BanWithReasonButtonCustomID   = "ban_with_reason_modal"
	OpenOutfitsMenuButtonCustomID = "open_outfits_menu"
	OpenFriendsMenuButtonCustomID = "open_friends_menu"
	OpenGroupsMenuButtonCustomID  = "open_groups_menu"

	BackButtonCustomID  = "back"
	BanButtonCustomID   = "ban"
	ClearButtonCustomID = "clear"
	SkipButtonCustomID  = "skip"

	// Friends Menu.
	FriendsPerPage     = 21
	FriendsGridColumns = 3
	FriendsGridRows    = 7

	// Outfits Menu.
	OutfitsPerPage    = 15
	OutfitGridColumns = 3
	OutfitGridRows    = 5

	// Groups Menu.
	GroupsPerPage     = 15
	GroupsGridColumns = 3
	GroupsGridRows    = 5

	// User Settings.
	UserSettingPrefix   = "user"
	UserSettingSelectID = "user_setting_select"
	StreamerModeOption  = "streamer_mode"
	DefaultSortOption   = "default_sort"

	// Guild Settings.
	GuildSettingPrefix     = "guild"
	GuildSettingSelectID   = "guild_setting_select"
	WhitelistedRolesOption = "whitelisted_roles"

	// Logs Menu.
	LogsPerPage                         = 10
	LogQueryBrowserCustomID             = "log_query_browser"
	LogsQueryUserIDOption               = "query_user_id_modal"
	LogsQueryReviewerIDOption           = "query_reviewer_id_modal"
	LogsQueryIDInputCustomID            = "query_id_input"
	LogsQueryActivityTypeFilterCustomID = "activity_type_filter"

	// Session keys.
	SessionKeyMessageID   = "messageID"
	SessionKeyTarget      = "target"
	SessionKeySortBy      = "sortBy"
	SessionKeyCurrentPage = "currentPage"

	SessionKeyFile         = "file"
	SessionKeyFileName     = "fileName"
	SessionKeyStreamerMode = "streamerMode"

	SessionKeyPaginationPage = "paginationPage"
	SessionKeyStart          = "start"
	SessionKeyTotalItems     = "totalItems"

	SessionKeyFlaggedCount   = "flaggedCount"
	SessionKeyConfirmedCount = "confirmedCount"

	SessionKeySettingName      = "settingName"
	SessionKeySettingType      = "settingType"
	SessionKeyUserSettings     = "userSettings"
	SessionKeyGuildSettings    = "guildSettings"
	SessionKeyCurrentValueFunc = "currentValueFunc"
	SessionKeyCustomID         = "customID"
	SessionKeyOptions          = "options"
	SessionKeyRoles            = "roles"

	SessionKeyFriends        = "friends"
	SessionKeyFlaggedFriends = "flaggedFriends"

	SessionKeyGroups        = "groups"
	SessionKeyFlaggedGroups = "flaggedGroups"

	SessionKeyOutfits = "outfits"

	SessionKeyLogs               = "logs"
	SessionKeyQueryType          = "queryType"
	SessionKeyQueryID            = "queryID"
	SessionKeyActivityTypeFilter = "activityTypeFilter"
)