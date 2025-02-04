// Code generated by "enumer -type=ActivityType -trimprefix=ActivityType"; DO NOT EDIT.

package enum

import (
	"fmt"
	"strings"
)

const _ActivityTypeName = "AllUserViewedUserLookupUserConfirmedUserClearedUserSkippedUserRecheckedUserTrainingUpvoteUserTrainingDownvoteUserDeletedGroupViewedGroupLookupGroupConfirmedGroupConfirmedCustomGroupClearedGroupSkippedGroupTrainingUpvoteGroupTrainingDownvoteGroupDeletedAppealSubmittedAppealSkippedAppealAcceptedAppealRejectedAppealClosedDiscordUserBannedDiscordUserUnbannedBotSettingUpdated"

var _ActivityTypeIndex = [...]uint16{0, 3, 13, 23, 36, 47, 58, 71, 89, 109, 120, 131, 142, 156, 176, 188, 200, 219, 240, 252, 267, 280, 294, 308, 320, 337, 356, 373}

const _ActivityTypeLowerName = "alluservieweduserlookupuserconfirmedusercleareduserskippeduserrecheckedusertrainingupvoteusertrainingdownvoteuserdeletedgroupviewedgrouplookupgroupconfirmedgroupconfirmedcustomgroupclearedgroupskippedgrouptrainingupvotegrouptrainingdownvotegroupdeletedappealsubmittedappealskippedappealacceptedappealrejectedappealcloseddiscorduserbanneddiscorduserunbannedbotsettingupdated"

func (i ActivityType) String() string {
	if i < 0 || i >= ActivityType(len(_ActivityTypeIndex)-1) {
		return fmt.Sprintf("ActivityType(%d)", i)
	}
	return _ActivityTypeName[_ActivityTypeIndex[i]:_ActivityTypeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _ActivityTypeNoOp() {
	var x [1]struct{}
	_ = x[ActivityTypeAll-(0)]
	_ = x[ActivityTypeUserViewed-(1)]
	_ = x[ActivityTypeUserLookup-(2)]
	_ = x[ActivityTypeUserConfirmed-(3)]
	_ = x[ActivityTypeUserCleared-(4)]
	_ = x[ActivityTypeUserSkipped-(5)]
	_ = x[ActivityTypeUserRechecked-(6)]
	_ = x[ActivityTypeUserTrainingUpvote-(7)]
	_ = x[ActivityTypeUserTrainingDownvote-(8)]
	_ = x[ActivityTypeUserDeleted-(9)]
	_ = x[ActivityTypeGroupViewed-(10)]
	_ = x[ActivityTypeGroupLookup-(11)]
	_ = x[ActivityTypeGroupConfirmed-(12)]
	_ = x[ActivityTypeGroupConfirmedCustom-(13)]
	_ = x[ActivityTypeGroupCleared-(14)]
	_ = x[ActivityTypeGroupSkipped-(15)]
	_ = x[ActivityTypeGroupTrainingUpvote-(16)]
	_ = x[ActivityTypeGroupTrainingDownvote-(17)]
	_ = x[ActivityTypeGroupDeleted-(18)]
	_ = x[ActivityTypeAppealSubmitted-(19)]
	_ = x[ActivityTypeAppealSkipped-(20)]
	_ = x[ActivityTypeAppealAccepted-(21)]
	_ = x[ActivityTypeAppealRejected-(22)]
	_ = x[ActivityTypeAppealClosed-(23)]
	_ = x[ActivityTypeDiscordUserBanned-(24)]
	_ = x[ActivityTypeDiscordUserUnbanned-(25)]
	_ = x[ActivityTypeBotSettingUpdated-(26)]
}

var _ActivityTypeValues = []ActivityType{ActivityTypeAll, ActivityTypeUserViewed, ActivityTypeUserLookup, ActivityTypeUserConfirmed, ActivityTypeUserCleared, ActivityTypeUserSkipped, ActivityTypeUserRechecked, ActivityTypeUserTrainingUpvote, ActivityTypeUserTrainingDownvote, ActivityTypeUserDeleted, ActivityTypeGroupViewed, ActivityTypeGroupLookup, ActivityTypeGroupConfirmed, ActivityTypeGroupConfirmedCustom, ActivityTypeGroupCleared, ActivityTypeGroupSkipped, ActivityTypeGroupTrainingUpvote, ActivityTypeGroupTrainingDownvote, ActivityTypeGroupDeleted, ActivityTypeAppealSubmitted, ActivityTypeAppealSkipped, ActivityTypeAppealAccepted, ActivityTypeAppealRejected, ActivityTypeAppealClosed, ActivityTypeDiscordUserBanned, ActivityTypeDiscordUserUnbanned, ActivityTypeBotSettingUpdated}

var _ActivityTypeNameToValueMap = map[string]ActivityType{
	_ActivityTypeName[0:3]:          ActivityTypeAll,
	_ActivityTypeLowerName[0:3]:     ActivityTypeAll,
	_ActivityTypeName[3:13]:         ActivityTypeUserViewed,
	_ActivityTypeLowerName[3:13]:    ActivityTypeUserViewed,
	_ActivityTypeName[13:23]:        ActivityTypeUserLookup,
	_ActivityTypeLowerName[13:23]:   ActivityTypeUserLookup,
	_ActivityTypeName[23:36]:        ActivityTypeUserConfirmed,
	_ActivityTypeLowerName[23:36]:   ActivityTypeUserConfirmed,
	_ActivityTypeName[36:47]:        ActivityTypeUserCleared,
	_ActivityTypeLowerName[36:47]:   ActivityTypeUserCleared,
	_ActivityTypeName[47:58]:        ActivityTypeUserSkipped,
	_ActivityTypeLowerName[47:58]:   ActivityTypeUserSkipped,
	_ActivityTypeName[58:71]:        ActivityTypeUserRechecked,
	_ActivityTypeLowerName[58:71]:   ActivityTypeUserRechecked,
	_ActivityTypeName[71:89]:        ActivityTypeUserTrainingUpvote,
	_ActivityTypeLowerName[71:89]:   ActivityTypeUserTrainingUpvote,
	_ActivityTypeName[89:109]:       ActivityTypeUserTrainingDownvote,
	_ActivityTypeLowerName[89:109]:  ActivityTypeUserTrainingDownvote,
	_ActivityTypeName[109:120]:      ActivityTypeUserDeleted,
	_ActivityTypeLowerName[109:120]: ActivityTypeUserDeleted,
	_ActivityTypeName[120:131]:      ActivityTypeGroupViewed,
	_ActivityTypeLowerName[120:131]: ActivityTypeGroupViewed,
	_ActivityTypeName[131:142]:      ActivityTypeGroupLookup,
	_ActivityTypeLowerName[131:142]: ActivityTypeGroupLookup,
	_ActivityTypeName[142:156]:      ActivityTypeGroupConfirmed,
	_ActivityTypeLowerName[142:156]: ActivityTypeGroupConfirmed,
	_ActivityTypeName[156:176]:      ActivityTypeGroupConfirmedCustom,
	_ActivityTypeLowerName[156:176]: ActivityTypeGroupConfirmedCustom,
	_ActivityTypeName[176:188]:      ActivityTypeGroupCleared,
	_ActivityTypeLowerName[176:188]: ActivityTypeGroupCleared,
	_ActivityTypeName[188:200]:      ActivityTypeGroupSkipped,
	_ActivityTypeLowerName[188:200]: ActivityTypeGroupSkipped,
	_ActivityTypeName[200:219]:      ActivityTypeGroupTrainingUpvote,
	_ActivityTypeLowerName[200:219]: ActivityTypeGroupTrainingUpvote,
	_ActivityTypeName[219:240]:      ActivityTypeGroupTrainingDownvote,
	_ActivityTypeLowerName[219:240]: ActivityTypeGroupTrainingDownvote,
	_ActivityTypeName[240:252]:      ActivityTypeGroupDeleted,
	_ActivityTypeLowerName[240:252]: ActivityTypeGroupDeleted,
	_ActivityTypeName[252:267]:      ActivityTypeAppealSubmitted,
	_ActivityTypeLowerName[252:267]: ActivityTypeAppealSubmitted,
	_ActivityTypeName[267:280]:      ActivityTypeAppealSkipped,
	_ActivityTypeLowerName[267:280]: ActivityTypeAppealSkipped,
	_ActivityTypeName[280:294]:      ActivityTypeAppealAccepted,
	_ActivityTypeLowerName[280:294]: ActivityTypeAppealAccepted,
	_ActivityTypeName[294:308]:      ActivityTypeAppealRejected,
	_ActivityTypeLowerName[294:308]: ActivityTypeAppealRejected,
	_ActivityTypeName[308:320]:      ActivityTypeAppealClosed,
	_ActivityTypeLowerName[308:320]: ActivityTypeAppealClosed,
	_ActivityTypeName[320:337]:      ActivityTypeDiscordUserBanned,
	_ActivityTypeLowerName[320:337]: ActivityTypeDiscordUserBanned,
	_ActivityTypeName[337:356]:      ActivityTypeDiscordUserUnbanned,
	_ActivityTypeLowerName[337:356]: ActivityTypeDiscordUserUnbanned,
	_ActivityTypeName[356:373]:      ActivityTypeBotSettingUpdated,
	_ActivityTypeLowerName[356:373]: ActivityTypeBotSettingUpdated,
}

var _ActivityTypeNames = []string{
	_ActivityTypeName[0:3],
	_ActivityTypeName[3:13],
	_ActivityTypeName[13:23],
	_ActivityTypeName[23:36],
	_ActivityTypeName[36:47],
	_ActivityTypeName[47:58],
	_ActivityTypeName[58:71],
	_ActivityTypeName[71:89],
	_ActivityTypeName[89:109],
	_ActivityTypeName[109:120],
	_ActivityTypeName[120:131],
	_ActivityTypeName[131:142],
	_ActivityTypeName[142:156],
	_ActivityTypeName[156:176],
	_ActivityTypeName[176:188],
	_ActivityTypeName[188:200],
	_ActivityTypeName[200:219],
	_ActivityTypeName[219:240],
	_ActivityTypeName[240:252],
	_ActivityTypeName[252:267],
	_ActivityTypeName[267:280],
	_ActivityTypeName[280:294],
	_ActivityTypeName[294:308],
	_ActivityTypeName[308:320],
	_ActivityTypeName[320:337],
	_ActivityTypeName[337:356],
	_ActivityTypeName[356:373],
}

// ActivityTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func ActivityTypeString(s string) (ActivityType, error) {
	if val, ok := _ActivityTypeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _ActivityTypeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to ActivityType values", s)
}

// ActivityTypeValues returns all values of the enum
func ActivityTypeValues() []ActivityType {
	return _ActivityTypeValues
}

// ActivityTypeStrings returns a slice of all String values of the enum
func ActivityTypeStrings() []string {
	strs := make([]string, len(_ActivityTypeNames))
	copy(strs, _ActivityTypeNames)
	return strs
}

// IsAActivityType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i ActivityType) IsAActivityType() bool {
	for _, v := range _ActivityTypeValues {
		if i == v {
			return true
		}
	}
	return false
}
