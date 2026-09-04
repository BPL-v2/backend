package service

import (
	"bpl/repository"
	"bpl/scoring"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ==================== Mock Implementations ====================

// mockSignupService implements SignupService
type mockSignupService struct{ mock.Mock }

func (m *mockSignupService) SaveSignup(signup *repository.Signup) (*repository.Signup, error) {
	args := m.Called(signup)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Signup), args.Error(1)
}
func (m *mockSignupService) RemoveSignupForUser(userId int, eventId int) error {
	return m.Called(userId, eventId).Error(0)
}
func (m *mockSignupService) GetSignupForUser(userId int, eventId int) (*repository.Signup, error) {
	args := m.Called(userId, eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Signup), args.Error(1)
}
func (m *mockSignupService) ReportPlaytime(userId int, eventId int, actualPlaytime int) (*repository.Signup, error) {
	args := m.Called(userId, eventId, actualPlaytime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Signup), args.Error(1)
}
func (m *mockSignupService) GetSignupsForEvent(event *repository.Event) ([]*repository.Signup, error) {
	args := m.Called(event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Signup), args.Error(1)
}
func (m *mockSignupService) GetExtendedSignupsForEvent(event *repository.Event) ([]*repository.Signup, map[int]map[int]time.Duration, map[int]map[int]int, error) {
	args := m.Called(event)
	return args.Get(0).([]*repository.Signup), args.Get(1).(map[int]map[int]time.Duration), args.Get(2).(map[int]map[int]int), args.Error(3)
}

// mockTeamService implements TeamService
type mockTeamService struct{ mock.Mock }

func (m *mockTeamService) GetTeamsForEvent(eventId int) ([]*repository.Team, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Team), args.Error(1)
}
func (m *mockTeamService) SaveTeam(team *repository.Team) (*repository.Team, error) {
	args := m.Called(team)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Team), args.Error(1)
}
func (m *mockTeamService) GetTeamById(teamId int) (*repository.Team, error) {
	args := m.Called(teamId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Team), args.Error(1)
}
func (m *mockTeamService) DeleteTeam(teamId int) error {
	return m.Called(teamId).Error(0)
}
func (m *mockTeamService) AddUsersToTeams(teamUsers []*repository.TeamUser, event *repository.Event) error {
	return m.Called(teamUsers, event).Error(0)
}
func (m *mockTeamService) GetTeamUsersForEvent(eventId int) ([]*repository.TeamUser, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.TeamUser), args.Error(1)
}
func (m *mockTeamService) GetTeamUserMapForEvent(event *repository.Event) (*map[int]int, error) {
	args := m.Called(event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*map[int]int), args.Error(1)
}
func (m *mockTeamService) GetTeamForUser(eventId int, userId int) (*repository.TeamUser, error) {
	args := m.Called(eventId, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.TeamUser), args.Error(1)
}
func (m *mockTeamService) GetTeamLeadsForEvent(eventId int) (map[int][]*repository.TeamUser, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int][]*repository.TeamUser), args.Error(1)
}
func (m *mockTeamService) GetSortedUsersForEvent(eventId int) ([]*SortedUser, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*SortedUser), args.Error(1)
}

// mockTeamRepo implements repository.TeamRepository
type mockTeamRepo struct{ mock.Mock }

func (m *mockTeamRepo) GetTeamsForEvent(eventId int) ([]*repository.Team, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Team), args.Error(1)
}
func (m *mockTeamRepo) Save(team *repository.Team) (*repository.Team, error) {
	args := m.Called(team)
	return args.Get(0).(*repository.Team), args.Error(1)
}
func (m *mockTeamRepo) GetTeamById(teamId int) (*repository.Team, error) {
	args := m.Called(teamId)
	return args.Get(0).(*repository.Team), args.Error(1)
}
func (m *mockTeamRepo) Delete(teamId int) error {
	return m.Called(teamId).Error(0)
}
func (m *mockTeamRepo) RemoveTeamUsersForEvent(teamUsers []*repository.TeamUser, event *repository.Event) error {
	return m.Called(teamUsers, event).Error(0)
}
func (m *mockTeamRepo) AddUsersToTeams(teamUsers []*repository.TeamUser) error {
	return m.Called(teamUsers).Error(0)
}
func (m *mockTeamRepo) GetTeamUsersForEvent(eventId int) ([]*repository.TeamUser, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.TeamUser), args.Error(1)
}
func (m *mockTeamRepo) GetTeamForUser(eventId int, userId int) (*repository.TeamUser, error) {
	args := m.Called(eventId, userId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.TeamUser), args.Error(1)
}
func (m *mockTeamRepo) GetTeamLeadsForEvent(eventId int) ([]*repository.TeamUser, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.TeamUser), args.Error(1)
}
func (m *mockTeamRepo) GetTeamUsersForTeam(teamId int) ([]*repository.TeamUser, error) {
	args := m.Called(teamId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.TeamUser), args.Error(1)
}
func (m *mockTeamRepo) RemoveUserForEvent(userId int, eventId int) error {
	return m.Called(userId, eventId).Error(0)
}
func (m *mockTeamRepo) GetAllTeamUsers() ([]*repository.TeamUser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.TeamUser), args.Error(1)
}
func (m *mockTeamRepo) GetAllTeamLeadUserIds() ([]int, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int), args.Error(1)
}
func (m *mockTeamRepo) GetNumbersOfPastEventsParticipatedByUsers(userIds []int) (map[int]int, error) {
	args := m.Called(userIds)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int]int), args.Error(1)
}

// mockUserRepo implements repository.UserRepository
type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) GetUserById(userId int, preloads ...string) (*repository.User, error) {
	args := m.Called(userId, preloads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.User), args.Error(1)
}
func (m *mockUserRepo) GetUsersByIds(userIds []int, preloads ...string) ([]*repository.User, error) {
	args := m.Called(userIds, preloads)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.User), args.Error(1)
}
func (m *mockUserRepo) SaveUser(user *repository.User) (*repository.User, error) {
	args := m.Called(user)
	return args.Get(0).(*repository.User), args.Error(1)
}
func (m *mockUserRepo) GetAllUsers() ([]*repository.User, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.User), args.Error(1)
}
func (m *mockUserRepo) GetStreamersForEvent(eventId int) ([]*repository.Streamer, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Streamer), args.Error(1)
}
func (m *mockUserRepo) GetUsersForEvent(eventId int) ([]*repository.TeamUserWithPoEToken, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.TeamUserWithPoEToken), args.Error(1)
}
func (m *mockUserRepo) GetUsersWithTeamForEvent(eventId int) (map[int]*repository.UserWithTeam, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int]*repository.UserWithTeam), args.Error(1)
}

// mockOauthRepo implements repository.OauthRepository
type mockOauthRepo struct{ mock.Mock }

func (m *mockOauthRepo) GetOauthForTokenRefresh(provider repository.Provider) (*repository.Oauth, error) {
	args := m.Called(provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Oauth), args.Error(1)
}
func (m *mockOauthRepo) GetOauthByProviderAndAccountId(provider repository.Provider, accountId string) (*repository.Oauth, error) {
	args := m.Called(provider, accountId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Oauth), args.Error(1)
}
func (m *mockOauthRepo) GetOauthByProviderAndAccountName(provider repository.Provider, accountName string) (*repository.Oauth, error) {
	args := m.Called(provider, accountName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Oauth), args.Error(1)
}
func (m *mockOauthRepo) GetAllOauths() ([]*repository.Oauth, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Oauth), args.Error(1)
}
func (m *mockOauthRepo) DeleteOauthsByUserIdAndProvider(userId int, provider repository.Provider) error {
	return m.Called(userId, provider).Error(0)
}
func (m *mockOauthRepo) SaveOauth(oauth *repository.Oauth) (*repository.Oauth, error) {
	args := m.Called(oauth)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Oauth), args.Error(1)
}

// ==================== Pure Function Tests: Activity ====================

func TestDetermineActiveTime_SingleActivity(t *testing.T) {
	now := time.Now()
	activities := []*repository.Activity{{Time: now}}
	result := determineActiveTime(activities, 5*time.Minute)
	assert.Equal(t, time.Duration(0), result, "single activity should have zero duration")
}

func TestDetermineActiveTime_ContinuousSession(t *testing.T) {
	now := time.Now()
	activities := []*repository.Activity{
		{Time: now},
		{Time: now.Add(1 * time.Minute)},
		{Time: now.Add(2 * time.Minute)},
		{Time: now.Add(4 * time.Minute)},
	}
	result := determineActiveTime(activities, 5*time.Minute)
	assert.Equal(t, 4*time.Minute, result, "all within threshold should be one session")
}

func TestDetermineActiveTime_MultipleSessions(t *testing.T) {
	now := time.Now()
	activities := []*repository.Activity{
		// Session 1: 0-3 min (3 min duration)
		{Time: now},
		{Time: now.Add(3 * time.Minute)},
		// Gap of 10 min (> 5 min threshold)
		// Session 2: 13-15 min (2 min duration)
		{Time: now.Add(13 * time.Minute)},
		{Time: now.Add(15 * time.Minute)},
	}
	result := determineActiveTime(activities, 5*time.Minute)
	assert.Equal(t, 5*time.Minute, result, "should be 3min + 2min = 5min")
}

func TestDetermineActiveTime_UnsortedInput(t *testing.T) {
	now := time.Now()
	activities := []*repository.Activity{
		{Time: now.Add(10 * time.Minute)},
		{Time: now},
		{Time: now.Add(5 * time.Minute)},
	}
	result := determineActiveTime(activities, 20*time.Minute)
	assert.Equal(t, 10*time.Minute, result, "should sort and compute correctly")
}

// ==================== Pure Function Tests: StashChange ====================

func TestChangeIdToInt_Valid(t *testing.T) {
	assert.Equal(t, 600, ChangeIdToInt("100-200-300"))
	assert.Equal(t, 42, ChangeIdToInt("42"))
}

func TestChangeIdToInt_Invalid(t *testing.T) {
	assert.Equal(t, 0, ChangeIdToInt("abc-def"))
	assert.Equal(t, 0, ChangeIdToInt(""))
}

// ==================== Pure Function Tests: Score Trie ====================

func TestBuildTrieAndFindObjectiveId(t *testing.T) {
	objMap := map[int]string{
		1: "Mirror of Kalandra",
		2: "Headhunter",
		3: "Mageblood",
	}
	trie := buildTrie(objMap)

	// Exact match
	result := findObjectiveId("Mirror of Kalandra", trie)
	require.NotNil(t, result)
	assert.Equal(t, 1, *result)

	// Substring match (item name contains objective name)
	result = findObjectiveId("Superior Headhunter", trie)
	require.NotNil(t, result)
	assert.Equal(t, 2, *result)

	// No match
	result = findObjectiveId("Some Random Item", trie)
	assert.Nil(t, result)
}

func TestFindObjectiveId_EmptyTrie(t *testing.T) {
	trie := buildTrie(map[int]string{})
	result := findObjectiveId("anything", trie)
	assert.Nil(t, result)
}

func TestFindObjectiveId_PartialMatch(t *testing.T) {
	trie := buildTrie(map[int]string{10: "Mirror"})

	result := findObjectiveId("Mirr", trie)
	assert.Nil(t, result, "partial match should not return result")

	result = findObjectiveId("Mirror", trie)
	require.NotNil(t, result)
	assert.Equal(t, 10, *result)
}

// ==================== Pure Function Tests: Score Diff ====================

func TestGetScoreDifference_New(t *testing.T) {
	score := &scoring.Score{
		ObjectiveId:       1,
		TeamId:            1,
		PresetCompletions: map[int]*scoring.PresetCompletion{},
	}
	diff := GetScoreDifference(nil, score)
	assert.Equal(t, Added, diff.DiffType)
	assert.Equal(t, score, diff.Score)
}

func TestGetScoreDifference_Unchanged(t *testing.T) {
	score := &scoring.Score{
		ObjectiveId: 1,
		TeamId:      1,
		PresetCompletions: map[int]*scoring.PresetCompletion{
			100: {Points: 10, Rank: 1, Number: 5, Finished: true},
		},
	}
	prev := &ScoreDifference{Score: score, DiffType: Unchanged}

	diff := GetScoreDifference(prev, score)
	assert.Equal(t, Unchanged, diff.DiffType)
}

func TestGetScoreDifference_Changed(t *testing.T) {
	oldScore := &scoring.Score{
		ObjectiveId: 1,
		TeamId:      1,
		PresetCompletions: map[int]*scoring.PresetCompletion{
			100: {Points: 10, Rank: 1, Number: 5, Finished: false},
		},
	}
	newScore := &scoring.Score{
		ObjectiveId: 1,
		TeamId:      1,
		PresetCompletions: map[int]*scoring.PresetCompletion{
			100: {Points: 20, Rank: 2, Number: 5, Finished: true},
		},
	}
	prev := &ScoreDifference{Score: oldScore}
	diff := GetScoreDifference(prev, newScore)
	assert.Equal(t, Changed, diff.DiffType)
	assert.Contains(t, diff.FieldDiff, "Points")
	assert.Contains(t, diff.FieldDiff, "Rank")
	assert.Contains(t, diff.FieldDiff, "Finished")
	assert.NotContains(t, diff.FieldDiff, "Number")
}

func TestDiff_AddedRemovedChanged(t *testing.T) {
	// Old scores: team1/obj1 and team1/obj2
	oldMap := make(ScoreMap)
	oldScore1 := &scoring.Score{ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{
		100: {Points: 10},
	}}
	oldScore2 := &scoring.Score{ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{
		100: {Points: 5},
	}}
	oldMap.setDiff(oldScore1, &ScoreDifference{Score: oldScore1})
	oldMap.setDiff(oldScore2, &ScoreDifference{Score: oldScore2})

	// New scores: team1/obj1 (changed) and team1/obj3 (added), obj2 removed
	newScores := []*scoring.Score{
		{ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{
			100: {Points: 20},
		}},
		{ObjectiveId: 3, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{
			100: {Points: 15},
		}},
	}

	newMap, diffMap := Diff(oldMap, newScores)

	// newMap should have obj1 and obj3
	assert.Contains(t, newMap[1], 1)
	assert.Contains(t, newMap[1], 3)
	assert.NotContains(t, newMap[1], 2)

	// diffMap should have: obj1 changed, obj2 removed, obj3 added
	assert.Equal(t, Changed, diffMap[1][1].DiffType)
	assert.Equal(t, Removed, diffMap[1][2].DiffType)
	assert.Equal(t, Added, diffMap[1][3].DiffType)
}

func TestScoreMap_GetSimpleScore(t *testing.T) {
	sm := make(ScoreMap)
	sm.setDiff(
		&scoring.Score{ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{100: {Points: 10}}},
		&ScoreDifference{Score: &scoring.Score{ObjectiveId: 1, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{100: {Points: 10}}}},
	)
	sm.setDiff(
		&scoring.Score{ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{100: {Points: 5}}},
		&ScoreDifference{Score: &scoring.Score{ObjectiveId: 2, TeamId: 1, PresetCompletions: map[int]*scoring.PresetCompletion{100: {Points: 5}}}},
	)
	sm.setDiff(
		&scoring.Score{ObjectiveId: 1, TeamId: 2, PresetCompletions: map[int]*scoring.PresetCompletion{100: {Points: 7}}},
		&ScoreDifference{Score: &scoring.Score{ObjectiveId: 1, TeamId: 2, PresetCompletions: map[int]*scoring.PresetCompletion{100: {Points: 7}}}},
	)

	simple := sm.GetSimpleScore()
	assert.Equal(t, 15, simple[1])
	assert.Equal(t, 7, simple[2])
}

// ==================== Mock-Based Tests: EventService ====================

func TestGetEventStatus_NoUser(t *testing.T) {
	mockSignup := new(mockSignupService)
	mockTeam := new(mockTeamService)

	event := &repository.Event{Id: 1, MaxSize: 10}
	mockSignup.On("GetSignupsForEvent", event).Return([]*repository.Signup{
		{UserId: 10}, {UserId: 20},
	}, nil)

	svc := &EventServiceImpl{signupService: mockSignup, teamService: mockTeam}
	status, err := svc.GetEventStatus(event, nil)
	require.NoError(t, err)
	assert.Equal(t, ApplicationStatusNone, status.ApplicationStatus)
	assert.Equal(t, 2, status.NumberOfSignups)
}

func TestGetEventStatus_Applied(t *testing.T) {
	mockSignup := new(mockSignupService)
	mockTeam := new(mockTeamService)

	event := &repository.Event{Id: 1, MaxSize: 10}
	user := &repository.User{Id: 5}

	mockSignup.On("GetSignupsForEvent", event).Return([]*repository.Signup{
		{UserId: 1, User: &repository.User{Id: 1}},
		{UserId: 2, User: &repository.User{Id: 2}},
		{UserId: 5, User: &repository.User{Id: 5}},
	}, nil)
	mockTeam.On("GetTeamForUser", 1, 5).Return(nil, gorm.ErrRecordNotFound)

	svc := &EventServiceImpl{signupService: mockSignup, teamService: mockTeam}
	status, err := svc.GetEventStatus(event, user)
	require.NoError(t, err)
	assert.Equal(t, ApplicationStatusApplied, status.ApplicationStatus)
	assert.Equal(t, 2, status.NumberOfSignupsBefore)
}

func TestGetEventStatus_Waitlisted(t *testing.T) {
	mockSignup := new(mockSignupService)
	mockTeam := new(mockTeamService)

	event := &repository.Event{Id: 1, MaxSize: 2} // Only 2 slots
	user := &repository.User{Id: 5}

	mockSignup.On("GetSignupsForEvent", event).Return([]*repository.Signup{
		{UserId: 1, User: &repository.User{Id: 1}},
		{UserId: 2, User: &repository.User{Id: 2}},
		{UserId: 5, User: &repository.User{Id: 5}}, // 3rd signup, beyond MaxSize=2
	}, nil)
	mockTeam.On("GetTeamForUser", 1, 5).Return(nil, gorm.ErrRecordNotFound)

	svc := &EventServiceImpl{signupService: mockSignup, teamService: mockTeam}
	status, err := svc.GetEventStatus(event, user)
	require.NoError(t, err)
	assert.Equal(t, ApplicationStatusWaitlisted, status.ApplicationStatus)
}

func TestGetEventStatus_Accepted(t *testing.T) {
	mockSignup := new(mockSignupService)
	mockTeam := new(mockTeamService)

	event := &repository.Event{Id: 1, MaxSize: 10}
	user := &repository.User{Id: 5}

	mockSignup.On("GetSignupsForEvent", event).Return([]*repository.Signup{
		{UserId: 5, User: &repository.User{Id: 5}},
	}, nil)
	mockTeam.On("GetTeamForUser", 1, 5).Return(&repository.TeamUser{
		TeamId: 42, UserId: 5, IsTeamLead: true,
	}, nil)

	svc := &EventServiceImpl{signupService: mockSignup, teamService: mockTeam}
	status, err := svc.GetEventStatus(event, user)
	require.NoError(t, err)
	assert.Equal(t, ApplicationStatusAccepted, status.ApplicationStatus)
	assert.NotNil(t, status.TeamId)
	assert.Equal(t, 42, *status.TeamId)
	assert.True(t, status.IsTeamLead)
}

func TestGetEventStatus_PartnerWish(t *testing.T) {
	mockSignup := new(mockSignupService)
	mockTeam := new(mockTeamService)

	event := &repository.Event{Id: 1, MaxSize: 10}
	poeName := "MyPoEAccount#1234"
	user := &repository.User{
		Id:            5,
		OauthAccounts: []*repository.Oauth{{Provider: repository.ProviderPoE, Name: poeName}},
	}

	partnerWish := "MyPoEAccount"
	mockSignup.On("GetSignupsForEvent", event).Return([]*repository.Signup{
		{
			UserId:      10,
			PartnerWish: &partnerWish,
			User: &repository.User{
				Id: 10,
				OauthAccounts: []*repository.Oauth{
					{Provider: repository.ProviderPoE, Name: "PartnerAccount#999"},
				},
			},
		},
		{UserId: 5, User: &repository.User{Id: 5}},
	}, nil)
	mockTeam.On("GetTeamForUser", 1, 5).Return(nil, gorm.ErrRecordNotFound)

	svc := &EventServiceImpl{signupService: mockSignup, teamService: mockTeam}
	status, err := svc.GetEventStatus(event, user)
	require.NoError(t, err)
	assert.Len(t, status.UsersWhoWantToSignUpWithYou, 1)
	assert.Equal(t, "PartnerAccount#999", status.UsersWhoWantToSignUpWithYou[0])
}

// ==================== Mock-Based Tests: TeamService ====================

func TestGetSortedUsersForEvent(t *testing.T) {
	mockTeamR := new(mockTeamRepo)
	mockUserR := new(mockUserRepo)

	mockTeamR.On("GetTeamUsersForEvent", 1).Return([]*repository.TeamUser{
		{TeamId: 10, UserId: 100, IsTeamLead: true},
		{TeamId: 10, UserId: 101, IsTeamLead: false},
		{TeamId: 20, UserId: 102, IsTeamLead: false},
	}, nil)

	mockUserR.On("GetUsersByIds", []int{100, 101, 102}, mock.Anything).Return([]*repository.User{
		{
			Id:          100,
			DisplayName: "Alice",
			OauthAccounts: []*repository.Oauth{
				{Provider: repository.ProviderPoE, Name: "alice_poe"},
				{Provider: repository.ProviderDiscord, Name: "alice_discord", AccountId: "disc100"},
			},
		},
		{
			Id:          101,
			DisplayName: "Bob",
			OauthAccounts: []*repository.Oauth{
				{Provider: repository.ProviderPoE, Name: "bob_poe"},
			},
		},
		{
			Id:          102,
			DisplayName: "Charlie",
			OauthAccounts: []*repository.Oauth{
				{Provider: repository.ProviderDiscord, Name: "charlie_discord", AccountId: "disc102"},
			},
		},
	}, nil)

	svc := &TeamServiceImpl{teamRepository: mockTeamR, userRepository: mockUserR}
	sorted, err := svc.GetSortedUsersForEvent(1)
	require.NoError(t, err)
	assert.Len(t, sorted, 3)

	// Alice
	assert.Equal(t, "Alice", sorted[0].DisplayName)
	assert.Equal(t, "alice_poe", sorted[0].PoEName)
	assert.Equal(t, "alice_discord", sorted[0].DiscordName)
	assert.Equal(t, "disc100", sorted[0].DiscordId)
	assert.Equal(t, 10, sorted[0].TeamId)
	assert.True(t, sorted[0].IsTeamLead)

	// Bob - no discord
	assert.Equal(t, "bob_poe", sorted[1].PoEName)
	assert.Equal(t, "", sorted[1].DiscordName)

	// Charlie - no PoE
	assert.Equal(t, "", sorted[2].PoEName)
	assert.Equal(t, "charlie_discord", sorted[2].DiscordName)
}

func TestGetTeamLeadsForEvent(t *testing.T) {
	mockTeamR := new(mockTeamRepo)

	mockTeamR.On("GetTeamLeadsForEvent", 1).Return([]*repository.TeamUser{
		{TeamId: 10, UserId: 100, IsTeamLead: true},
		{TeamId: 10, UserId: 101, IsTeamLead: true},
		{TeamId: 20, UserId: 200, IsTeamLead: true},
	}, nil)

	svc := &TeamServiceImpl{teamRepository: mockTeamR}
	leads, err := svc.GetTeamLeadsForEvent(1)
	require.NoError(t, err)
	assert.Len(t, leads[10], 2, "team 10 should have 2 leads")
	assert.Len(t, leads[20], 1, "team 20 should have 1 lead")
}

func TestGetTeamUserMapForEvent(t *testing.T) {
	mockTeamR := new(mockTeamRepo)

	mockTeamR.On("GetTeamUsersForEvent", 1).Return([]*repository.TeamUser{
		{TeamId: 10, UserId: 100},
		{TeamId: 10, UserId: 101},
		{TeamId: 20, UserId: 200},
	}, nil)

	svc := &TeamServiceImpl{teamRepository: mockTeamR}
	userMap, err := svc.GetTeamUserMapForEvent(&repository.Event{Id: 1})
	require.NoError(t, err)
	assert.Equal(t, 10, (*userMap)[100])
	assert.Equal(t, 10, (*userMap)[101])
	assert.Equal(t, 20, (*userMap)[200])
}

// ==================== Mock-Based Tests: UserService ====================

func TestGetAllUsers_WithOauthPreload(t *testing.T) {
	mockUR := new(mockUserRepo)
	mockOR := new(mockOauthRepo)

	mockUR.On("GetAllUsers").Return([]*repository.User{
		{Id: 1, DisplayName: "user1"},
		{Id: 2, DisplayName: "user2"},
		{Id: 3, DisplayName: "user3"},
	}, nil)

	mockOR.On("GetAllOauths").Return([]*repository.Oauth{
		{UserId: 1, Provider: repository.ProviderPoE, Name: "poe1"},
		{UserId: 1, Provider: repository.ProviderDiscord, Name: "disc1"},
		{UserId: 3, Provider: repository.ProviderPoE, Name: "poe3"},
	}, nil)

	svc := &UserServiceImpl{userRepository: mockUR, oauthRepository: mockOR}
	users, err := svc.GetAllUsers("OauthAccounts")
	require.NoError(t, err)
	assert.Len(t, users, 3)

	assert.Len(t, users[0].OauthAccounts, 2, "user1 should have 2 oauth accounts")
	assert.Nil(t, users[1].OauthAccounts, "user2 should have no oauth accounts")
	assert.Len(t, users[2].OauthAccounts, 1, "user3 should have 1 oauth account")
}

func TestGetAllUsers_WithoutPreload(t *testing.T) {
	mockUR := new(mockUserRepo)
	mockOR := new(mockOauthRepo)

	mockUR.On("GetAllUsers").Return([]*repository.User{
		{Id: 1, DisplayName: "user1"},
	}, nil)

	svc := &UserServiceImpl{userRepository: mockUR, oauthRepository: mockOR}
	users, err := svc.GetAllUsers()
	require.NoError(t, err)
	assert.Len(t, users, 1)
	mockOR.AssertNotCalled(t, "GetAllOauths")
}

func TestRemoveProvider_LastProvider(t *testing.T) {
	svc := &UserServiceImpl{}
	user := &repository.User{
		Id:            1,
		OauthAccounts: []*repository.Oauth{{Provider: repository.ProviderPoE}},
	}

	_, err := svc.RemoveProvider(user, repository.ProviderPoE)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove last provider")
}

func TestRemoveProvider_Success(t *testing.T) {
	mockUR := new(mockUserRepo)
	mockOR := new(mockOauthRepo)

	user := &repository.User{
		Id: 1,
		OauthAccounts: []*repository.Oauth{
			{Provider: repository.ProviderPoE, UserId: 1},
			{Provider: repository.ProviderDiscord, UserId: 1},
		},
	}
	mockOR.On("DeleteOauthsByUserIdAndProvider", 1, repository.ProviderPoE).Return(nil)
	mockUR.On("GetUserById", 1, []string{"OauthAccounts"}).Return(&repository.User{
		Id: 1,
		OauthAccounts: []*repository.Oauth{
			{Provider: repository.ProviderDiscord, UserId: 1},
		},
	}, nil)

	svc := &UserServiceImpl{userRepository: mockUR, oauthRepository: mockOR}
	updated, err := svc.RemoveProvider(user, repository.ProviderPoE)
	require.NoError(t, err)
	assert.Len(t, updated.OauthAccounts, 1)
	assert.Equal(t, repository.ProviderDiscord, updated.OauthAccounts[0].Provider)
	mockOR.AssertCalled(t, "DeleteOauthsByUserIdAndProvider", 1, repository.ProviderPoE)
}

// ==================== HTTP Mock Test: GetNinjaChangeId ====================

func TestGetNinjaChangeId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := NinjaResponse{NextChangeId: "123-456-789"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Override package-level URL
	original := ninjaStatsURL
	ninjaStatsURL = server.URL
	defer func() { ninjaStatsURL = original }()

	changeId, err := GetNinjaChangeId()
	require.NoError(t, err)
	assert.Equal(t, "123-456-789", changeId)
}

func TestGetNinjaChangeId_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	original := ninjaStatsURL
	ninjaStatsURL = server.URL
	defer func() { ninjaStatsURL = original }()

	_, err := GetNinjaChangeId()
	assert.Error(t, err)
}

// ==================== Achievement Helper Tests ====================

func TestHasLevelNCharacter(t *testing.T) {
	chars := []*repository.Character{
		{Level: 80},
		{Level: 95},
		{Level: 70},
	}

	assert.True(t, hasLevelNCharacter(90, chars))
	assert.True(t, hasLevelNCharacter(95, chars))
	assert.False(t, hasLevelNCharacter(96, chars))
	assert.False(t, hasLevelNCharacter(100, chars))
}

func TestHasLevelNCharacter_Empty(t *testing.T) {
	assert.False(t, hasLevelNCharacter(1, []*repository.Character{}))
}

func TestPlayedNLeagues(t *testing.T) {
	chars := make([]*repository.Character, 5)
	assert.True(t, playedNLeagues(1, chars))
	assert.True(t, playedNLeagues(5, chars))
	assert.False(t, playedNLeagues(6, chars))
}

func TestPlayedNDifferentAscendancies(t *testing.T) {
	chars := []*repository.Character{
		{Ascendancy: "Berserker"},
		{Ascendancy: "Deadeye"},
		{Ascendancy: "Necromancer"},
		{Ascendancy: "Berserker"}, // duplicate
		{Ascendancy: "Marauder"},  // base class - should be excluded
		{Ascendancy: "Witch"},     // base class - should be excluded
	}

	assert.True(t, playedNDifferentAscendancies(3, chars))
	assert.False(t, playedNDifferentAscendancies(4, chars))
}

func TestPlayedNDifferentAscendancies_AllBaseClasses(t *testing.T) {
	chars := []*repository.Character{
		{Ascendancy: "Scion"},
		{Ascendancy: "Marauder"},
		{Ascendancy: "Ranger"},
	}
	assert.False(t, playedNDifferentAscendancies(1, chars))
}

func TestAchievementChecks_CharacterBased(t *testing.T) {
	chars := make([]*repository.Character, 10)
	for i := range chars {
		chars[i] = &repository.Character{Level: 95, Ascendancy: "Asc" + string(rune('A'+i))}
	}
	characterMap := map[int][]*repository.Character{1: chars}

	assertGrants := func(key repository.AchievementCheckKey, wantGranted bool) {
		userIds, err := achievementChecks[key](nil, nil, characterMap)
		assert.NoError(t, err)
		if wantGranted {
			assert.Contains(t, userIds, 1)
		} else {
			assert.NotContains(t, userIds, 1)
		}
	}
	assertGrants(repository.CheckLevel90, true)
	assertGrants(repository.CheckLevel95, true)
	assertGrants(repository.CheckLevel100, false)
	assertGrants(repository.CheckParticipatedInEvent, true)
	assertGrants(repository.CheckPlayed5Leagues, true)
	assertGrants(repository.CheckPlayed10Leagues, true)
	assertGrants(repository.CheckPlayed10Ascendancies, true)
}

func TestAchievementChecks_CharacterBased_MinimalChars(t *testing.T) {
	characterMap := map[int][]*repository.Character{
		1: {{Level: 50, Ascendancy: "Berserker"}},
	}

	userIds, err := achievementChecks[repository.CheckParticipatedInEvent](nil, nil, characterMap)
	assert.NoError(t, err)
	assert.Contains(t, userIds, 1)

	userIds, err = achievementChecks[repository.CheckLevel90](nil, nil, characterMap)
	assert.NoError(t, err)
	assert.NotContains(t, userIds, 1)

	userIds, err = achievementChecks[repository.CheckPlayed5Leagues](nil, nil, characterMap)
	assert.NoError(t, err)
	assert.NotContains(t, userIds, 1)
}

func TestAchievementChecks_CharacterBased_EventScoped(t *testing.T) {
	characterMap := map[int][]*repository.Character{
		1: {
			{Level: 95, EventId: 1},
			{Level: 10, EventId: 2},
		},
	}

	eventTwo := 2
	userIds, err := achievementChecks[repository.CheckLevel90](nil, &eventTwo, characterMap)
	assert.NoError(t, err)
	assert.NotContains(t, userIds, 1, "user's only level-90 character is in a different event")

	eventOne := 1
	userIds, err = achievementChecks[repository.CheckLevel90](nil, &eventOne, characterMap)
	assert.NoError(t, err)
	assert.Contains(t, userIds, 1)
}

func TestAchievementChecks_Teamlead_AllEvents(t *testing.T) {
	mockTeamRepo := new(mockTeamRepo)
	svc := &AchievementServiceImpl{teamRepository: mockTeamRepo}

	mockTeamRepo.On("GetAllTeamLeadUserIds").Return([]int{10, 20}, nil)

	userIds, err := achievementChecks[repository.CheckTeamlead](svc, nil, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{10, 20}, userIds)
	mockTeamRepo.AssertExpectations(t)
}

func TestAchievementChecks_Teamlead_EventScoped(t *testing.T) {
	mockTeamRepo := new(mockTeamRepo)
	svc := &AchievementServiceImpl{teamRepository: mockTeamRepo}

	eventId := 1
	mockTeamRepo.On("GetTeamLeadsForEvent", eventId).Return([]*repository.TeamUser{{UserId: 10}, {UserId: 20}}, nil)

	userIds, err := achievementChecks[repository.CheckTeamlead](svc, &eventId, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{10, 20}, userIds)
	mockTeamRepo.AssertExpectations(t)
}

func TestAchievementChecks_SubmittedBounty_EventScoped(t *testing.T) {
	mockSubmissionRepo := new(mockSubmissionRepository)
	svc := &AchievementServiceImpl{submissionRepository: mockSubmissionRepo}

	eventId := 1
	mockSubmissionRepo.On(
		"GetApprovedSubmissionUserIds",
		mock.MatchedBy(func(id *int) bool { return id != nil && *id == eventId }),
	).Return([]int{10}, nil)

	userIds, err := achievementChecks[repository.CheckSubmittedBounty](svc, &eventId, nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{10}, userIds)
	mockSubmissionRepo.AssertExpectations(t)
}

// ==================== CharacterInfo.ToPlayerUpdate Tests ====================

func TestCharacterInfo_ToPlayerUpdate_ValidToken(t *testing.T) {
	ci := &CharacterInfo{
		User: &repository.User{
			Id: 42,
			OauthAccounts: []*repository.Oauth{
				{Provider: repository.ProviderPoE, AccessToken: "valid-token", Expiry: time.Now().Add(time.Hour), Name: "player#1234"},
			},
		},
		Event:     &repository.Event{Id: 1},
		Character: &repository.Character{Id: "char-1", Name: "TestChar", Ascendancy: "Berserker", Level: 90},
		TeamId:    5,
	}

	pu, err := ci.ToPlayerUpdate()
	require.NoError(t, err)
	assert.Equal(t, 42, pu.UserId)
	assert.Equal(t, 5, pu.TeamId)
	assert.Equal(t, "valid-token", pu.Token)
	assert.Equal(t, "char-1", pu.New.Character.Id)
	assert.Equal(t, "Berserker", pu.New.Character.Class)
}

func TestCharacterInfo_ToPlayerUpdate_ExpiredToken(t *testing.T) {
	ci := &CharacterInfo{
		User: &repository.User{
			Id: 42,
			OauthAccounts: []*repository.Oauth{
				{Provider: repository.ProviderPoE, AccessToken: "expired", Expiry: time.Now().Add(-time.Hour), Name: "player#1234"},
			},
		},
		Event:     &repository.Event{Id: 1},
		Character: &repository.Character{Id: "char-1", Name: "TestChar"},
		TeamId:    5,
	}

	_, err := ci.ToPlayerUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid PoE oauth token")
}

func TestCharacterInfo_ToPlayerUpdate_NoPoEAccount(t *testing.T) {
	ci := &CharacterInfo{
		User: &repository.User{
			Id: 42,
			OauthAccounts: []*repository.Oauth{
				{Provider: "discord", AccessToken: "discord-token", Expiry: time.Now().Add(time.Hour)},
			},
		},
		Event:     &repository.Event{Id: 1},
		Character: &repository.Character{Id: "char-1", Name: "TestChar"},
		TeamId:    5,
	}

	_, err := ci.ToPlayerUpdate()
	assert.Error(t, err)
}

// ==================== Submission Service Tests ====================

type mockSubmissionRepository struct{ mock.Mock }

func (m *mockSubmissionRepository) GetSubmissionsForEvent(event *repository.Event) ([]*repository.Submission, error) {
	args := m.Called(event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Submission), args.Error(1)
}
func (m *mockSubmissionRepository) GetSubmissionsForObjectives(objectives []*repository.Objective) ([]*repository.Submission, error) {
	args := m.Called(objectives)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*repository.Submission), args.Error(1)
}
func (m *mockSubmissionRepository) GetSubmissionById(id int) (*repository.Submission, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Submission), args.Error(1)
}
func (m *mockSubmissionRepository) SaveSubmission(submission *repository.Submission) (*repository.Submission, error) {
	args := m.Called(submission)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Submission), args.Error(1)
}
func (m *mockSubmissionRepository) AddMatchToSubmission(submission *repository.Submission) error {
	return m.Called(submission).Error(0)
}
func (m *mockSubmissionRepository) RemoveMatchFromSubmission(submission *repository.Submission) error {
	return m.Called(submission).Error(0)
}
func (m *mockSubmissionRepository) GetApprovedSubmissionUserIds(eventId *int) ([]int, error) {
	args := m.Called(eventId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int), args.Error(1)
}
func (m *mockSubmissionRepository) DeleteSubmission(submissionId int) error {
	return m.Called(submissionId).Error(0)
}

func TestSubmissionService_SaveSubmission_New(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	submission := &repository.Submission{ObjectiveId: 1, Proof: "screenshot.png"}
	submitter := &repository.User{Id: 10}

	mockRepo.On("SaveSubmission", mock.AnythingOfType("*repository.Submission")).Return(
		&repository.Submission{Id: 1, ObjectiveId: 1, UserId: 10, ApprovalStatus: repository.PENDING}, nil,
	)

	result, err := svc.SaveSubmission(submission, submitter)
	require.NoError(t, err)
	assert.Equal(t, repository.PENDING, result.ApprovalStatus)
	assert.Equal(t, 10, submission.UserId)
	mockRepo.AssertExpectations(t)
}

func TestSubmissionService_SaveSubmission_UpdateOwn(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	existing := &repository.Submission{Id: 5, ObjectiveId: 1, UserId: 10, ApprovalStatus: repository.APPROVED}
	submission := &repository.Submission{Id: 5, ObjectiveId: 2, Proof: "new-proof.png"}
	submitter := &repository.User{Id: 10}

	mockRepo.On("GetSubmissionById", 5).Return(existing, nil)
	mockRepo.On("RemoveMatchFromSubmission", existing).Return(nil)
	mockRepo.On("SaveSubmission", existing).Return(existing, nil)

	result, err := svc.SaveSubmission(submission, submitter)
	require.NoError(t, err)
	assert.Equal(t, repository.PENDING, result.ApprovalStatus) // was APPROVED, should become PENDING
	assert.Equal(t, 2, existing.ObjectiveId)
	mockRepo.AssertExpectations(t)
}

func TestSubmissionService_SaveSubmission_UpdateOther_Forbidden(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	existing := &repository.Submission{Id: 5, ObjectiveId: 1, UserId: 10}
	submission := &repository.Submission{Id: 5, ObjectiveId: 2}
	submitter := &repository.User{Id: 99} // different user

	mockRepo.On("GetSubmissionById", 5).Return(existing, nil)

	_, err := svc.SaveSubmission(submission, submitter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestSubmissionService_ReviewSubmission_Approve(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	existing := &repository.Submission{Id: 1, ObjectiveId: 1, UserId: 10, ApprovalStatus: repository.PENDING}
	reviewer := &repository.User{Id: 99}
	review := &repository.Submission{ApprovalStatus: repository.APPROVED}

	mockRepo.On("GetSubmissionById", 1).Return(existing, nil)
	mockRepo.On("AddMatchToSubmission", existing).Return(nil)
	mockRepo.On("SaveSubmission", existing).Return(existing, nil)

	result, err := svc.ReviewSubmission(1, review, reviewer)
	require.NoError(t, err)
	assert.Equal(t, repository.APPROVED, result.ApprovalStatus)
	assert.Equal(t, &reviewer.Id, result.ReviewerId)
	mockRepo.AssertExpectations(t)
}

func TestSubmissionService_ReviewSubmission_Reject(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	existing := &repository.Submission{Id: 1, ObjectiveId: 1, UserId: 10, ApprovalStatus: repository.APPROVED}
	reviewer := &repository.User{Id: 99}
	reviewComment := "insufficient proof"
	review := &repository.Submission{ApprovalStatus: repository.REJECTED, ReviewComment: &reviewComment}

	mockRepo.On("GetSubmissionById", 1).Return(existing, nil)
	mockRepo.On("RemoveMatchFromSubmission", existing).Return(nil)
	mockRepo.On("SaveSubmission", existing).Return(existing, nil)

	result, err := svc.ReviewSubmission(1, review, reviewer)
	require.NoError(t, err)
	assert.Equal(t, repository.REJECTED, result.ApprovalStatus)
	assert.Equal(t, &reviewComment, result.ReviewComment)
	mockRepo.AssertExpectations(t)
}

func TestSubmissionService_SaveBulkSubmissions(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	submissions := []*repository.Submission{
		{ObjectiveId: 1, Proof: "p1"},
		{ObjectiveId: 2, Proof: "p2"},
	}

	mockRepo.On("SaveSubmission", submissions[0]).Return(&repository.Submission{Id: 1, ObjectiveId: 1}, nil)
	mockRepo.On("SaveSubmission", submissions[1]).Return(&repository.Submission{Id: 2, ObjectiveId: 2}, nil)

	result, err := svc.SaveBulkSubmissions(submissions)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestSubmissionService_DeleteSubmission(t *testing.T) {
	mockRepo := new(mockSubmissionRepository)
	svc := &SubmissionServiceImpl{submissionRepository: mockRepo}

	mockRepo.On("DeleteSubmission", 5).Return(nil)

	err := svc.DeleteSubmission(&repository.Submission{Id: 5}, &repository.User{Id: 1})
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
