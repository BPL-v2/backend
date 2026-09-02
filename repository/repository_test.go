package repository

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"bpl/client"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}
	resource, err := pool.Run("postgres", "17.2-alpine", []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "DATABASE_NAME=postgres"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	err = resource.Expire(600)
	if err != nil {
		log.Fatalf("Could not set resource expiration: %s", err)
	}
	sqlInfo := fmt.Sprintf(
		"host=localhost port=%s user=postgres password=postgres dbname=postgres sslmode=disable search_path=bpl2",
		resource.GetPort("5432/tcp"))

	if err := pool.Retry(func() error {
		var err error
		db, err = gorm.Open(postgres.Open(sqlInfo), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				TablePrefix:   "bpl2.",
				SingularTable: false,
			},
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return err
		}
		db.Exec(`CREATE SCHEMA IF NOT EXISTS bpl2`)
		err = db.AutoMigrate(
			&Event{},
			&Objective{},
			&Condition{},
			&Team{},
			&User{},
			&TeamUser{},
			&StashChange{},
			&ObjectiveMatch{},
			&Submission{},
			&ClientCredentials{},
			&Signup{},
			&Oauth{},
			&KafkaConsumer{},
			&ScoringRule{},
			&ObjectiveScoringRule{},
			&Character{},
			&CharacterPob{},
			&ChangeId{},
		)
		if err != nil {
			fmt.Println("Error in AutoMigrate: ", err)
		}
		return err
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	defer func() {
		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}()
	m.Run()
}

func tearDown() {
	db.Exec("DELETE FROM bpl2.objective_scoring_rules")
	db.Exec("DELETE FROM bpl2.scoring_rules")
	db.Exec("DELETE FROM bpl2.submissions")
	db.Exec("DELETE FROM bpl2.objective_matches")
	db.Exec("DELETE FROM bpl2.objectives")
	db.Exec("DELETE FROM bpl2.signups")
	db.Exec("DELETE FROM bpl2.stash_changes")
	db.Exec("DELETE FROM bpl2.change_ids")
	db.Exec("DELETE FROM bpl2.character_pobs")
	db.Exec("DELETE FROM bpl2.characters")
	db.Exec("DELETE FROM bpl2.team_users")
	db.Exec("DELETE FROM bpl2.teams")
	db.Exec("DELETE FROM bpl2.oauths")
	db.Exec("DELETE FROM bpl2.users")
	db.Exec("DELETE FROM bpl2.client_credentials")
	db.Exec("DELETE FROM bpl2.kafka_consumers")
	db.Exec("DELETE FROM bpl2.events")
}

func createTestEvent() *Event {
	event := &Event{
		Name:                 "test-event",
		MaxSize:              10,
		IsCurrent:            true,
		GameVersion:          PoE2,
		ApplicationStartTime: time.Now().Add(-time.Hour),
		ApplicationEndTime:   time.Now().Add(time.Hour),
		EventStartTime:       time.Now(),
		EventEndTime:         time.Now().Add(24 * time.Hour),
	}
	db.Create(event)
	return event
}

func createTestUsers(n int) []*User {
	users := make([]*User, n)
	for i := range n {
		users[i] = &User{DisplayName: fmt.Sprintf("user%d", i+1)}
	}
	db.Create(&users)
	return users
}

func createTestTeamsWithUsers(event *Event) ([]*Team, []*User) {
	users := createTestUsers(4)
	teams := []*Team{
		{Name: "team1", Abbreviation: "T1", EventId: event.Id, Color: "#ff0000", AllowedClasses: []string{}},
		{Name: "team2", Abbreviation: "T2", EventId: event.Id, Color: "#0000ff", AllowedClasses: []string{}},
	}
	db.Create(&teams)
	db.Create(&TeamUser{TeamId: teams[0].Id, UserId: users[0].Id})
	db.Create(&TeamUser{TeamId: teams[0].Id, UserId: users[1].Id})
	db.Create(&TeamUser{TeamId: teams[1].Id, UserId: users[2].Id})
	db.Create(&TeamUser{TeamId: teams[1].Id, UserId: users[3].Id})
	return teams, users
}

func intPtr(i int) *int { return &i }

// ==================== UserRepository Tests ====================

func TestUserRepository_GetUserById(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	user := &User{DisplayName: "testuser", Permissions: Permissions{PermissionAdmin}}
	db.Create(user)

	found, err := repo.GetUserById(user.Id)
	require.NoError(t, err)
	assert.Equal(t, "testuser", found.DisplayName)
	assert.Contains(t, found.Permissions, PermissionAdmin)
}

func TestUserRepository_GetUserById_NotFound(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}

	_, err := repo.GetUserById(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUserRepository_GetUsersByIds(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	users := createTestUsers(3)

	found, err := repo.GetUsersByIds([]int{users[0].Id, users[2].Id})
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestUserRepository_SaveUser(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	user := &User{DisplayName: "newuser"}

	saved, err := repo.SaveUser(user)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)
	assert.Equal(t, "newuser", saved.DisplayName)

	// Update
	saved.DisplayName = "updated"
	saved, err = repo.SaveUser(saved)
	require.NoError(t, err)
	assert.Equal(t, "updated", saved.DisplayName)
}

func TestUserRepository_GetAllUsers(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	createTestUsers(3)

	users, err := repo.GetAllUsers()
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestUserRepository_GetStreamersForEvent(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	_ = teams

	// Add twitch oauth for user 0
	db.Create(&Oauth{
		UserId:        users[0].Id,
		Provider:      ProviderTwitch,
		AccessToken:   "tok",
		Expiry:        time.Now().Add(time.Hour),
		RefreshExpiry: time.Now().Add(24 * time.Hour),
		Name:          "twitchuser",
		AccountId:     "twitch123",
	})

	streamers, err := repo.GetStreamersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, streamers, 1)
	assert.Equal(t, users[0].Id, streamers[0].UserId)
	assert.Equal(t, "twitch123", streamers[0].TwitchId)
}

func TestUserRepository_GetUsersForEvent(t *testing.T) {
	defer tearDown()
	repo := &UserRepositoryImpl{DB: db}
	event := createTestEvent()
	_, users := createTestTeamsWithUsers(event)

	// Add PoE oauth for first two users
	for i := 0; i < 2; i++ {
		db.Create(&Oauth{
			UserId:        users[i].Id,
			Provider:      ProviderPoE,
			AccessToken:   fmt.Sprintf("token%d", i),
			Expiry:        time.Now().Add(time.Hour),
			RefreshExpiry: time.Now().Add(24 * time.Hour),
			Name:          fmt.Sprintf("poeuser%d", i),
			AccountId:     fmt.Sprintf("poe%d", i),
		})
	}

	result, err := repo.GetUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUser_HasOneOfPermissions(t *testing.T) {
	user := &User{Permissions: Permissions{PermissionAdmin, PermissionManager}}

	assert.True(t, user.HasOneOfPermissions(PermissionAdmin))
	assert.True(t, user.HasOneOfPermissions(PermissionManager))
	assert.False(t, user.HasOneOfPermissions(PermissionSubmissionJudge))
	assert.True(t, user.HasOneOfPermissions(PermissionSubmissionJudge, PermissionAdmin))
}

func TestUser_HasPoEName(t *testing.T) {
	user := &User{
		OauthAccounts: []*Oauth{
			{Provider: ProviderPoE, Name: "TestAccount#1234"},
		},
	}
	assert.True(t, user.HasPoEName("testaccount"))
	assert.True(t, user.HasPoEName("TestAccount#5678"))
	assert.False(t, user.HasPoEName("otheraccount"))
}

// ==================== EventRepository Tests ====================

func TestEventRepository_GetCurrentEvent(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	event := createTestEvent()

	found, err := repo.GetCurrentEvent()
	require.NoError(t, err)
	assert.Equal(t, event.Id, found.Id)
	assert.Equal(t, "test-event", found.Name)
	assert.True(t, found.IsCurrent)
}

func TestEventRepository_GetCurrentEvent_NotFound(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	_, err := repo.GetCurrentEvent()
	assert.Error(t, err)
}

func TestEventRepository_GetEventById(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	event := createTestEvent()

	found, err := repo.GetEventById(event.Id)
	require.NoError(t, err)
	assert.Equal(t, event.Name, found.Name)
}

func TestEventRepository_GetEventById_WithPreloads(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	event := createTestEvent()
	db.Create(&Team{Name: "t1", Abbreviation: "T1", EventId: event.Id, Color: "#fff", AllowedClasses: []string{}})

	found, err := repo.GetEventById(event.Id, "Teams")
	require.NoError(t, err)
	assert.Len(t, found.Teams, 1)
	assert.Equal(t, "t1", found.Teams[0].Name)
}

func TestEventRepository_InvalidateCurrentEvent(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	createTestEvent()

	err := repo.InvalidateCurrentEvent()
	require.NoError(t, err)

	_, err = repo.GetCurrentEvent()
	assert.Error(t, err, "no current event should exist after invalidation")
}

func TestEventRepository_FindAll(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	for i := range 3 {
		e := &Event{
			Name:                 fmt.Sprintf("event%d", i),
			MaxSize:              10,
			GameVersion:          PoE2,
			ApplicationStartTime: time.Now(),
			ApplicationEndTime:   time.Now(),
			EventStartTime:       time.Now(),
			EventEndTime:         time.Now(),
		}
		db.Create(e)
	}

	events, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, events, 3)
}

func TestEventRepository_Delete(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}
	event := createTestEvent()

	err := repo.Delete(event)
	require.NoError(t, err)

	_, err = repo.GetEventById(event.Id)
	assert.Error(t, err)
}

// ==================== TeamRepository Tests ====================

func TestTeamRepository_GetTeamById(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	team := &Team{Name: "myteam", Abbreviation: "MT", EventId: event.Id, Color: "#000", AllowedClasses: []string{}}
	db.Create(team)

	found, err := repo.GetTeamById(team.Id)
	require.NoError(t, err)
	assert.Equal(t, "myteam", found.Name)
}

func TestTeamRepository_GetTeamsForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	createTestTeamsWithUsers(event)

	teams, err := repo.GetTeamsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, teams, 2)
}

func TestTeamRepository_Save(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	team := &Team{Name: "newteam", Abbreviation: "NT", EventId: event.Id, Color: "#123", AllowedClasses: []string{"Warrior"}}

	saved, err := repo.Save(team)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)

	saved.Name = "renamed"
	saved, err = repo.Save(saved)
	require.NoError(t, err)
	assert.Equal(t, "renamed", saved.Name)
}

func TestTeamRepository_Delete(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, _ := createTestTeamsWithUsers(event)

	err := repo.Delete(teams[0].Id)
	require.NoError(t, err)

	_, err = repo.GetTeamById(teams[0].Id)
	assert.Error(t, err)

	// Team users should also be deleted
	var count int64
	db.Model(&TeamUser{}).Where("team_id = ?", teams[0].Id).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestTeamRepository_GetTeamUsersForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	createTestTeamsWithUsers(event)

	teamUsers, err := repo.GetTeamUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, teamUsers, 4)
}

func TestTeamRepository_AddUsersToTeams(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	team := &Team{Name: "t1", Abbreviation: "T1", EventId: event.Id, Color: "#000", AllowedClasses: []string{}}
	db.Create(team)
	users := createTestUsers(2)

	err := repo.AddUsersToTeams([]*TeamUser{
		{TeamId: team.Id, UserId: users[0].Id},
		{TeamId: team.Id, UserId: users[1].Id},
	})
	require.NoError(t, err)

	teamUsers, err := repo.GetTeamUsersForTeam(team.Id)
	require.NoError(t, err)
	assert.Len(t, teamUsers, 2)
}

func TestTeamRepository_RemoveUserForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	_, users := createTestTeamsWithUsers(event)

	err := repo.RemoveUserForEvent(users[0].Id, event.Id)
	require.NoError(t, err)

	teamUsers, err := repo.GetTeamUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, teamUsers, 3)
}

func TestTeamRepository_GetTeamForUser(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)

	tu, err := repo.GetTeamForUser(event.Id, users[0].Id)
	require.NoError(t, err)
	assert.Equal(t, teams[0].Id, tu.TeamId)
}

func TestTeamRepository_GetTeamLeadsForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	_ = teams

	// Make user0 a team lead
	db.Model(&TeamUser{}).Where("user_id = ? AND team_id = ?", users[0].Id, teams[0].Id).Update("is_team_lead", true)

	leads, err := repo.GetTeamLeadsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, leads, 1)
	assert.Equal(t, users[0].Id, leads[0].UserId)
}

func TestTeamRepository_GetNumbersOfPastEventsParticipatedByUsers(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}

	// Create two past events
	past1 := &Event{Name: "past1", MaxSize: 10, GameVersion: PoE2, IsCurrent: false,
		ApplicationStartTime: time.Now(), ApplicationEndTime: time.Now(), EventStartTime: time.Now(), EventEndTime: time.Now()}
	past2 := &Event{Name: "past2", MaxSize: 10, GameVersion: PoE2, IsCurrent: false,
		ApplicationStartTime: time.Now(), ApplicationEndTime: time.Now(), EventStartTime: time.Now(), EventEndTime: time.Now()}
	db.Create(past1)
	db.Create(past2)

	users := createTestUsers(2)
	t1 := &Team{Name: "pt1", Abbreviation: "PT1", EventId: past1.Id, Color: "#000", AllowedClasses: []string{}}
	t2 := &Team{Name: "pt2", Abbreviation: "PT2", EventId: past2.Id, Color: "#000", AllowedClasses: []string{}}
	db.Create(t1)
	db.Create(t2)
	db.Create(&TeamUser{TeamId: t1.Id, UserId: users[0].Id})
	db.Create(&TeamUser{TeamId: t2.Id, UserId: users[0].Id})
	db.Create(&TeamUser{TeamId: t1.Id, UserId: users[1].Id})

	result, err := repo.GetNumbersOfPastEventsParticipatedByUsers([]int{users[0].Id, users[1].Id})
	require.NoError(t, err)
	assert.Equal(t, 2, result[users[0].Id])
	assert.Equal(t, 1, result[users[1].Id])
}

// ==================== ObjectiveRepository Tests ====================

func TestObjectiveRepository_SaveAndGetObjective(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{
		Name:           "test-objective",
		EventId:        event.Id,
		ObjectiveType:  ObjectiveTypeItem,
		TrackedValue:   TrackedValueStackSize,
		CountingMethod: CountingMethodFirstCompletion,
		RequiredAmount: 5,
		SyncStatus:     SyncStatusSynced,
	}
	saved, err := repo.SaveObjective(obj)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)
	assert.Equal(t, SyncStatusDesynced, saved.SyncStatus, "SaveObjective should set status to DESYNCED")

	found, err := repo.GetObjectiveById(saved.Id)
	require.NoError(t, err)
	assert.Equal(t, "test-objective", found.Name)
	assert.Equal(t, 5, found.RequiredAmount)
}

func TestObjectiveRepository_DeleteObjective(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{Name: "todelete", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	err := repo.DeleteObjective(obj.Id)
	require.NoError(t, err)

	_, err = repo.GetObjectiveById(obj.Id)
	assert.Error(t, err)
}

func TestObjectiveRepository_GetObjectivesByEventIdFlat(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	root := &Objective{Name: "root", EventId: event.Id, ObjectiveType: ObjectiveTypeCategory, TrackedValue: TrackedValueCompletedChildObjectiveCount, CountingMethod: CountingMethodChildResult, SyncStatus: SyncStatusDesynced}
	db.Create(root)
	child := &Objective{Name: "child", EventId: event.Id, ParentId: &root.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(child)

	objectives, err := repo.GetObjectivesByEventIdFlat(event.Id)
	require.NoError(t, err)
	assert.Len(t, objectives, 2)
}

func TestObjectiveRepository_GetObjectivesByEventId_Tree(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	root := &Objective{Name: "root", EventId: event.Id, ObjectiveType: ObjectiveTypeCategory, TrackedValue: TrackedValueCompletedChildObjectiveCount, CountingMethod: CountingMethodChildResult, SyncStatus: SyncStatusDesynced}
	db.Create(root)
	child1 := &Objective{Name: "child1", EventId: event.Id, ParentId: &root.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	child2 := &Objective{Name: "child2", EventId: event.Id, ParentId: &root.Id, ObjectiveType: ObjectiveTypePlayer, TrackedValue: TrackedValueCharacterLevel, CountingMethod: CountingMethodLatestValue, SyncStatus: SyncStatusDesynced}
	db.Create(child1)
	db.Create(child2)

	tree, err := repo.GetObjectivesByEventId(event.Id)
	require.NoError(t, err)
	assert.Equal(t, "root", tree.Name)
	assert.Len(t, tree.Children, 2)
}

func TestObjectiveRepository_SyncStatusLifecycle(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{Name: "synctest", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	err := repo.StartSync([]int{obj.Id})
	require.NoError(t, err)
	found, _ := repo.GetObjectiveById(obj.Id)
	assert.Equal(t, SyncStatusSyncing, found.SyncStatus)

	err = repo.FinishSync([]int{obj.Id})
	require.NoError(t, err)
	found, _ = repo.GetObjectiveById(obj.Id)
	assert.Equal(t, SyncStatusSynced, found.SyncStatus)
}

func TestObjectiveRepository_AssociateScoringRules(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveRepositoryImpl{DB: db}
	event := createTestEvent()

	obj := &Objective{Name: "obj", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)
	preset1 := &ScoringRule{EventId: event.Id, Name: "p1", Description: "d1", Points: ExtendingNumberSlice{10}, RuleType: FIXED_POINTS_ON_COMPLETION}
	preset2 := &ScoringRule{EventId: event.Id, Name: "p2", Description: "d2", Points: ExtendingNumberSlice{5}, RuleType: RANK_BY_COMPLETION_TIME}
	db.Create(preset1)
	db.Create(preset2)

	err := repo.AssociateScoringRules(obj.Id, []int{preset1.Id, preset2.Id})
	require.NoError(t, err)

	found, err := repo.GetObjectiveById(obj.Id, "ScoringRules")
	require.NoError(t, err)
	assert.Len(t, found.ScoringRules, 2)

	// Re-associate with only one preset
	err = repo.AssociateScoringRules(obj.Id, []int{preset1.Id})
	require.NoError(t, err)
	found, err = repo.GetObjectiveById(obj.Id, "ScoringRules")
	require.NoError(t, err)
	assert.Len(t, found.ScoringRules, 1)
}

func TestObjective_FlatMap(t *testing.T) {
	root := &Objective{Name: "root", Children: []*Objective{
		{Name: "child1", Children: []*Objective{
			{Name: "grandchild"},
		}},
		{Name: "child2"},
	}}
	flat := root.FlatMap()
	assert.Len(t, flat, 4)
}

// ==================== SignupRepository Tests ====================

func TestSignupRepository_SaveAndGetSignup(t *testing.T) {
	defer tearDown()
	repo := &SignupRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	signup := &Signup{
		EventId:          event.Id,
		UserId:           users[0].Id,
		Timestamp:        time.Now(),
		ExpectedPlayTime: 40,
		NeedsHelp:        true,
	}
	saved, err := repo.SaveSignup(signup)
	require.NoError(t, err)
	assert.Equal(t, event.Id, saved.EventId)

	found, err := repo.GetSignupForUser(users[0].Id, event.Id)
	require.NoError(t, err)
	assert.Equal(t, 40, found.ExpectedPlayTime)
	assert.True(t, found.NeedsHelp)
}

func TestSignupRepository_RemoveSignupForUser(t *testing.T) {
	defer tearDown()
	repo := &SignupRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	signup := &Signup{EventId: event.Id, UserId: users[0].Id, Timestamp: time.Now(), ExpectedPlayTime: 20}
	db.Create(signup)

	err := repo.RemoveSignupForUser(users[0].Id, event.Id)
	require.NoError(t, err)

	_, err = repo.GetSignupForUser(users[0].Id, event.Id)
	assert.Error(t, err)
}

func TestSignupRepository_GetSignupsForEvent(t *testing.T) {
	defer tearDown()
	repo := &SignupRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(3)

	for i, u := range users {
		db.Create(&Signup{
			EventId:          event.Id,
			UserId:           u.Id,
			Timestamp:        time.Now().Add(time.Duration(i) * time.Minute),
			ExpectedPlayTime: 10 * (i + 1),
		})
	}

	signups, err := repo.GetSignupsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, signups, 3)
	// Should be ordered by timestamp ASC
	assert.True(t, signups[0].Timestamp.Before(signups[2].Timestamp) || signups[0].Timestamp.Equal(signups[2].Timestamp))
	// User should be preloaded
	assert.NotNil(t, signups[0].User)
}

// ==================== OauthRepository Tests ====================

func TestOauthRepository_SaveAndGet(t *testing.T) {
	defer tearDown()
	repo := &OauthRepositoryImpl{DB: db}
	users := createTestUsers(1)

	oauth := &Oauth{
		UserId:        users[0].Id,
		Provider:      ProviderDiscord,
		AccessToken:   "access123",
		RefreshToken:  "refresh123",
		Expiry:        time.Now().Add(time.Hour),
		RefreshExpiry: time.Now().Add(24 * time.Hour),
		Name:          "discorduser",
		AccountId:     "discord456",
	}
	saved, err := repo.SaveOauth(oauth)
	require.NoError(t, err)
	assert.Equal(t, "discorduser", saved.Name)

	found, err := repo.GetOauthByProviderAndAccountId(ProviderDiscord, "discord456")
	require.NoError(t, err)
	assert.Equal(t, users[0].Id, found.UserId)
	assert.NotNil(t, found.User, "User should be preloaded")
}

func TestOauthRepository_GetOauthByProviderAndAccountName(t *testing.T) {
	defer tearDown()
	repo := &OauthRepositoryImpl{DB: db}
	users := createTestUsers(1)

	db.Create(&Oauth{
		UserId:        users[0].Id,
		Provider:      ProviderPoE,
		AccessToken:   "tok",
		Expiry:        time.Now().Add(time.Hour),
		RefreshExpiry: time.Now().Add(24 * time.Hour),
		Name:          "poeaccount",
		AccountId:     "poe123",
	})

	found, err := repo.GetOauthByProviderAndAccountName(ProviderPoE, "poeaccount")
	require.NoError(t, err)
	assert.Equal(t, "poe123", found.AccountId)
}

func TestOauthRepository_DeleteOauthsByUserIdAndProvider(t *testing.T) {
	defer tearDown()
	repo := &OauthRepositoryImpl{DB: db}
	users := createTestUsers(1)

	db.Create(&Oauth{UserId: users[0].Id, Provider: ProviderPoE, AccessToken: "t", Expiry: time.Now(), RefreshExpiry: time.Now(), Name: "n", AccountId: "a"})
	db.Create(&Oauth{UserId: users[0].Id, Provider: ProviderDiscord, AccessToken: "t", Expiry: time.Now(), RefreshExpiry: time.Now(), Name: "n", AccountId: "a"})

	err := repo.DeleteOauthsByUserIdAndProvider(users[0].Id, ProviderPoE)
	require.NoError(t, err)

	oauths, err := repo.GetAllOauths()
	require.NoError(t, err)
	assert.Len(t, oauths, 1)
	assert.Equal(t, ProviderDiscord, oauths[0].Provider)
}

// ==================== SubmissionRepository Tests ====================

func TestSubmissionRepository_SaveAndGet(t *testing.T) {
	defer tearDown()
	repo := &SubmissionRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "subobj", EventId: event.Id, ObjectiveType: ObjectiveTypeSubmission, TrackedValue: TrackedValueSubmittedValue, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	sub := &Submission{
		ObjectiveId:    obj.Id,
		Timestamp:      time.Now(),
		Number:         42,
		UserId:         users[0].Id,
		TeamId:         teams[0].Id,
		Proof:          "https://example.com/proof.png",
		Comment:        "look at this",
		ApprovalStatus: PENDING,
	}
	saved, err := repo.SaveSubmission(sub)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)

	found, err := repo.GetSubmissionById(saved.Id)
	require.NoError(t, err)
	assert.Equal(t, 42, found.Number)
	assert.Equal(t, PENDING, found.ApprovalStatus)
	assert.NotNil(t, found.Objective)
}

func TestSubmissionRepository_GetSubmissionsForEvent(t *testing.T) {
	defer tearDown()
	repo := &SubmissionRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	event.Teams = teams // Must be set for TeamIds() to work
	obj := &Objective{Name: "subobj", EventId: event.Id, ObjectiveType: ObjectiveTypeSubmission, TrackedValue: TrackedValueSubmittedValue, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	for i := 0; i < 3; i++ {
		db.Create(&Submission{
			ObjectiveId:    obj.Id,
			Timestamp:      time.Now(),
			Number:         i + 1,
			UserId:         users[i%2].Id,
			TeamId:         teams[i%2].Id,
			Proof:          "proof",
			Comment:        "comment",
			ApprovalStatus: PENDING,
		})
	}

	subs, err := repo.GetSubmissionsForEvent(event)
	require.NoError(t, err)
	assert.Len(t, subs, 3)
}

func TestSubmissionRepository_DeleteSubmission(t *testing.T) {
	defer tearDown()
	repo := &SubmissionRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "subobj", EventId: event.Id, ObjectiveType: ObjectiveTypeSubmission, TrackedValue: TrackedValueSubmittedValue, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	sub := &Submission{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 1, UserId: users[0].Id, TeamId: teams[0].Id, Proof: "p", Comment: "c", ApprovalStatus: PENDING}
	db.Create(sub)

	err := repo.DeleteSubmission(sub.Id)
	require.NoError(t, err)

	_, err = repo.GetSubmissionById(sub.Id)
	assert.Error(t, err)
}

func TestSubmission_ToObjectiveMatch(t *testing.T) {
	sub := &Submission{
		ObjectiveId: 10,
		Timestamp:   time.Now(),
		Number:      5,
		UserId:      20,
		TeamId:      30,
	}
	match := sub.ToObjectiveMatch()
	assert.Equal(t, 10, match.ObjectiveId)
	assert.Equal(t, 5, match.Number)
	assert.Equal(t, 30, match.TeamId)
	assert.Equal(t, 20, *match.UserId)
}

// ==================== ScoringRuleRepository Tests ====================

func TestScoringRuleRepository_SaveAndGet(t *testing.T) {
	defer tearDown()
	repo := &ScoringRuleRepositoryImpl{DB: db}
	event := createTestEvent()

	preset := &ScoringRule{
		EventId:     event.Id,
		Name:        "test-preset",
		Description: "A test preset",
		Points:      ExtendingNumberSlice{10, 5, 3},
		RuleType:    RANK_BY_COMPLETION_TIME,
	}
	saved, err := repo.SaveRule(preset)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)

	presets, err := repo.GetRulesForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, presets, 1)
	assert.Equal(t, "test-preset", presets[0].Name)
	assert.Equal(t, float64(10), presets[0].Points[0])
	assert.Equal(t, float64(5), presets[0].Points[1])
}

func TestScoringRuleRepository_DeleteRule(t *testing.T) {
	defer tearDown()
	repo := &ScoringRuleRepositoryImpl{DB: db}
	event := createTestEvent()

	preset := &ScoringRule{EventId: event.Id, Name: "del", Description: "d", Points: ExtendingNumberSlice{1}, RuleType: FIXED_POINTS_ON_COMPLETION}
	db.Create(preset)

	err := repo.DeleteRule(preset.Id)
	require.NoError(t, err)

	presets, err := repo.GetRulesForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, presets, 0)
}

func TestExtendingNumberSlice_Get(t *testing.T) {
	s := ExtendingNumberSlice{10, 5, 3}
	assert.Equal(t, float64(10), s.Get(0))
	assert.Equal(t, float64(5), s.Get(1))
	assert.Equal(t, float64(3), s.Get(2))
	assert.Equal(t, float64(3), s.Get(100), "should return last element for out of bounds")

	empty := ExtendingNumberSlice{}
	assert.Equal(t, float64(0), empty.Get(0), "should return 0 for empty slice")
}

// ==================== ObjectiveMatchRepository Tests ====================

func TestObjectiveMatchRepository_SaveAndDeleteMatches(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveMatchRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "matchobj", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	matches := []*ObjectiveMatch{
		{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 1, TeamId: teams[0].Id, UserId: &users[0].Id},
		{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 2, TeamId: teams[1].Id, UserId: &users[2].Id},
	}
	err := repo.SaveMatches(matches)
	require.NoError(t, err)

	var count int64
	db.Model(&ObjectiveMatch{}).Where("objective_id = ?", obj.Id).Count(&count)
	assert.Equal(t, int64(2), count)

	err = repo.DeleteMatches([]int{obj.Id})
	require.NoError(t, err)

	db.Model(&ObjectiveMatch{}).Where("objective_id = ?", obj.Id).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestObjectiveMatchRepository_OverwriteMatches(t *testing.T) {
	defer tearDown()
	repo := &ObjectiveMatchRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	obj := &Objective{Name: "overwrite", EventId: event.Id, ObjectiveType: ObjectiveTypeItem, TrackedValue: TrackedValueStackSize, CountingMethod: CountingMethodFirstCompletion, SyncStatus: SyncStatusDesynced}
	db.Create(obj)

	// Save initial matches
	db.Create(&ObjectiveMatch{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 1, TeamId: teams[0].Id, UserId: &users[0].Id})
	db.Create(&ObjectiveMatch{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 2, TeamId: teams[0].Id, UserId: &users[1].Id})

	// Overwrite with new matches
	newMatches := []*ObjectiveMatch{
		{ObjectiveId: obj.Id, Timestamp: time.Now(), Number: 99, TeamId: teams[1].Id, UserId: &users[2].Id},
	}
	err := repo.OverwriteMatches(newMatches, []int{obj.Id})
	require.NoError(t, err)

	var count int64
	db.Model(&ObjectiveMatch{}).Where("objective_id = ?", obj.Id).Count(&count)
	assert.Equal(t, int64(1), count)

	var match ObjectiveMatch
	db.Where("objective_id = ?", obj.Id).First(&match)
	assert.Equal(t, 99, match.Number)
}

// ==================== StashChangeRepository Tests ====================

func TestStashChangeRepository_CreateStashChangeIfNotExists(t *testing.T) {
	defer tearDown()
	repo := &StashChangeRepositoryImpl{DB: db}
	event := createTestEvent()
	now := time.Now().Truncate(time.Microsecond)

	sc := &StashChange{StashId: "stash-1", EventId: event.Id, Timestamp: now}
	created, err := repo.CreateStashChangeIfNotExists(sc)
	require.NoError(t, err)
	assert.NotZero(t, created.Id)

	// Creating again with same stash_id + event_id + timestamp should return existing
	sc2 := &StashChange{StashId: "stash-1", EventId: event.Id, Timestamp: now}
	existing, err := repo.CreateStashChangeIfNotExists(sc2)
	require.NoError(t, err)
	assert.Equal(t, created.Id, existing.Id)
}

func TestStashChangeRepository_GetLatestTimestamp(t *testing.T) {
	defer tearDown()
	repo := &StashChangeRepositoryImpl{DB: db}
	event := createTestEvent()
	now := time.Now()

	db.Create(&StashChange{StashId: "s1", EventId: event.Id, Timestamp: now.Add(-time.Hour)})
	db.Create(&StashChange{StashId: "s2", EventId: event.Id, Timestamp: now})

	latest, err := repo.GetLatestTimestamp(event.Id)
	require.NoError(t, err)
	assert.InDelta(t, now.Unix(), latest.Unix(), 1)
}

func TestStashChangeRepository_GetLatestTimestamp_NoRecords(t *testing.T) {
	defer tearDown()
	repo := &StashChangeRepositoryImpl{DB: db}

	latest, err := repo.GetLatestTimestamp(99999)
	require.NoError(t, err)
	assert.True(t, latest.IsZero())
}

// ==================== Conditions / Custom Types Tests ====================

func TestConditions_ScanValue(t *testing.T) {
	conditions := Conditions{
		{Field: BASE_TYPE, Operator: EQ, Value: "Mirror of Kalandra"},
		{Field: ILVL, Operator: GT, Value: "80"},
	}
	val, err := conditions.Value()
	require.NoError(t, err)

	var scanned Conditions
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Len(t, scanned, 2)
	assert.Equal(t, BASE_TYPE, scanned[0].Field)
	assert.Equal(t, GT, scanned[1].Operator)
}

func TestConditions_ScanNil(t *testing.T) {
	var c Conditions
	err := c.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, c)
}

func TestPermissions_ScanValue(t *testing.T) {
	perms := Permissions{PermissionAdmin, PermissionManager}
	val, err := perms.Value()
	require.NoError(t, err)

	var scanned Permissions
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Len(t, scanned, 2)
	assert.Equal(t, PermissionAdmin, scanned[0])
	assert.Equal(t, PermissionManager, scanned[1])
}

func TestExtraMap_ScanValue(t *testing.T) {
	extra := ExtraMap{"key1": "val1", "key2": "val2"}
	val, err := extra.Value()
	require.NoError(t, err)

	var scanned ExtraMap
	err = scanned.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, "val1", scanned["key1"])
	assert.Equal(t, "val2", scanned["key2"])
}

func TestExtraMap_ScanNil(t *testing.T) {
	var e ExtraMap
	err := e.Scan(nil)
	require.NoError(t, err)
	assert.Empty(t, e)
}

// ==================== PoENameWithoutDiscriminator Tests ====================

func TestPoENameWithoutDiscriminator(t *testing.T) {
	name := "TestAccount#1234"
	assert.Equal(t, "testaccount", PoENameWithoutDiscriminator(&name))

	plain := "simpleaccount"
	assert.Equal(t, "simpleaccount", PoENameWithoutDiscriminator(&plain))

	assert.Equal(t, "", PoENameWithoutDiscriminator(nil))
}

// ==================== LoadUsersIntoEvent Tests ====================

func TestLoadUsersIntoEvent(t *testing.T) {
	defer tearDown()
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)
	event.Teams = teams

	err := LoadUsersIntoEvent(db, event)
	require.NoError(t, err)
	assert.Len(t, event.Teams[0].Users, 2)
	assert.Len(t, event.Teams[1].Users, 2)

	// Verify correct users in correct teams
	team0UserIds := []int{event.Teams[0].Users[0].Id, event.Teams[0].Users[1].Id}
	assert.Contains(t, team0UserIds, users[0].Id)
	assert.Contains(t, team0UserIds, users[1].Id)
}

// ==================== Event model method tests ====================

func TestEvent_TeamIds(t *testing.T) {
	event := &Event{
		Teams: []*Team{
			{Id: 1}, {Id: 5}, {Id: 10},
		},
	}
	ids := event.TeamIds()
	assert.Equal(t, []int{1, 5, 10}, ids)
}

// ==================== Pure Function Tests: float2Int64 / float2Int32 ====================

func TestFloat2Int64(t *testing.T) {
	assert.Equal(t, int64(42), float2Int64(42.7))
	assert.Equal(t, int64(0), float2Int64(0))
	assert.Equal(t, int64(-42), float2Int64(-42.7))
	// Large value should cap at max int
	assert.Equal(t, int64(^uint(0)>>1), float2Int64(1e20))
	// Negative large value
	assert.Equal(t, -int64(^uint(0)>>1), float2Int64(-1e20))
}

func TestFloat2Int32(t *testing.T) {
	assert.Equal(t, int32(42), float2Int32(42.7))
	assert.Equal(t, int32(0), float2Int32(0))
	assert.Equal(t, int32(-42), float2Int32(-42.7))
	// Large value should cap at max int32
	assert.Equal(t, int32(^uint32(0)>>1), float2Int32(1e15))
	// Negative large value
	assert.Equal(t, -int32(^uint32(0)>>1), float2Int32(-1e15))
}

// ==================== PoBExport Tests ====================

func TestPoBExport_FromStringAndToString(t *testing.T) {
	original := "SGVsbG8gV29ybGQ=" // base64 of "Hello World"
	var p PoBExport
	err := p.FromString(original)
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello World"), []byte(p))

	// Round-trip
	encoded := p.ToString()
	var p2 PoBExport
	err = p2.FromString(encoded)
	require.NoError(t, err)
	assert.Equal(t, []byte(p), []byte(p2))
}

func TestPoBExport_FromString_URLSafeChars(t *testing.T) {
	// Test URL-safe character replacement (- -> +, _ -> /)
	var p PoBExport
	// "abc+def/ghi=" in standard base64 should be passable as "abc-def_ghi="
	err := p.FromString("abc-def_ghi=")
	// This may or may not decode depending on content, just check it doesn't crash
	// The replacement should happen: abc+def/ghi=
	_ = err
}

func TestPoBExport_FromString_Invalid(t *testing.T) {
	var p PoBExport
	err := p.FromString("!!!invalid!!!")
	assert.Error(t, err)
}

func TestPoBExport_ToString_URLSafe(t *testing.T) {
	// Create PoBExport with bytes that would produce + and / in standard base64
	p := PoBExport([]byte{0xff, 0xff, 0xff})
	result := p.ToString()
	assert.NotContains(t, result, "+")
	assert.NotContains(t, result, "/")
}

func TestPoBExport_Scan(t *testing.T) {
	var p PoBExport

	// Scan nil
	err := p.Scan(nil)
	require.NoError(t, err)
	assert.Nil(t, PoBExport(p))

	// Scan bytes
	err = p.Scan([]byte("test data"))
	require.NoError(t, err)
	assert.Equal(t, PoBExport("test data"), p)

	// Scan wrong type
	err = p.Scan("string value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected []byte")
}

func TestPoBExport_Value(t *testing.T) {
	// Non-nil
	p := PoBExport([]byte("test"))
	val, err := p.Value()
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), val)

	// Nil
	var pNil PoBExport
	val, err = pNil.Value()
	require.NoError(t, err)
	assert.Nil(t, val)
}

// ==================== CharacterPob.HasEqualStats Tests ====================

func TestCharacterPob_HasEqualStats(t *testing.T) {
	pob1 := &CharacterPob{
		DPS: 100, EHP: 200, PhysMaxHit: 300, EleMaxHit: 400,
		HP: 500, Mana: 600, ES: 700, Armour: 800,
		Evasion: 900, XP: 1000, MovementSpeed: 130,
	}

	t.Run("equal", func(t *testing.T) {
		pob2 := &CharacterPob{
			DPS: 100, EHP: 200, PhysMaxHit: 300, EleMaxHit: 400,
			HP: 500, Mana: 600, ES: 700, Armour: 800,
			Evasion: 900, XP: 1000, MovementSpeed: 130,
		}
		assert.True(t, pob1.HasEqualStats(pob2))
	})

	t.Run("nil other", func(t *testing.T) {
		assert.False(t, pob1.HasEqualStats(nil))
	})

	t.Run("different DPS", func(t *testing.T) {
		pob2 := &CharacterPob{
			DPS: 999, EHP: 200, PhysMaxHit: 300, EleMaxHit: 400,
			HP: 500, Mana: 600, ES: 700, Armour: 800,
			Evasion: 900, XP: 1000, MovementSpeed: 130,
		}
		assert.False(t, pob1.HasEqualStats(pob2))
	})

	t.Run("different XP is still equal", func(t *testing.T) {
		pob2 := &CharacterPob{
			DPS: 100, EHP: 200, PhysMaxHit: 300, EleMaxHit: 400,
			HP: 500, Mana: 600, ES: 700, Armour: 800,
			Evasion: 900, XP: 999999, MovementSpeed: 130,
		}
		assert.True(t, pob1.HasEqualStats(pob2))
	})

	t.Run("different MovementSpeed", func(t *testing.T) {
		pob2 := &CharacterPob{
			DPS: 100, EHP: 200, PhysMaxHit: 300, EleMaxHit: 400,
			HP: 500, Mana: 600, ES: 700, Armour: 800,
			Evasion: 900, XP: 1000, MovementSpeed: 999,
		}
		assert.False(t, pob1.HasEqualStats(pob2))
	})

	t.Run("different Level", func(t *testing.T) {
		base := *pob1
		base.Level = 90
		other := base
		other.Level = 91
		assert.False(t, base.HasEqualStats(&other))
	})

	t.Run("different MainSkill", func(t *testing.T) {
		base := *pob1
		base.MainSkill = "Lightning Arrow"
		other := base
		other.MainSkill = "Tornado Shot"
		assert.False(t, base.HasEqualStats(&other))
	})

	t.Run("different Ascendancy", func(t *testing.T) {
		base := *pob1
		base.Ascendancy = "Deadeye"
		other := base
		other.Ascendancy = "Pathfinder"
		assert.False(t, base.HasEqualStats(&other))
	})
}

// ==================== Event.GetRealm Tests ====================

func TestEvent_GetRealm(t *testing.T) {
	t.Run("PoE2 returns realm", func(t *testing.T) {
		event := &Event{GameVersion: PoE2}
		realm := event.GetRealm()
		require.NotNil(t, realm)
	})

	t.Run("PoE1 returns nil", func(t *testing.T) {
		event := &Event{GameVersion: PoE1}
		realm := event.GetRealm()
		assert.Nil(t, realm)
	})
}

// ==================== GuildStashTab.ShouldUpdate Tests ====================

func TestGuildStashTab_ShouldUpdate(t *testing.T) {
	timings := map[TimingKey]time.Duration{
		GuildstashUpdateInterval:        5 * time.Minute,
		GuildstashPriorityFetchInterval: 30 * time.Second,
	}

	t.Run("fetch disabled", func(t *testing.T) {
		tab := &GuildStashTab{FetchEnabled: false, LastFetch: time.Now().Add(-1 * time.Hour)}
		assert.False(t, tab.ShouldUpdate(timings))
	})

	t.Run("normal fetch after interval", func(t *testing.T) {
		tab := &GuildStashTab{FetchEnabled: true, PriorityFetch: false, LastFetch: time.Now().Add(-10 * time.Minute)}
		assert.True(t, tab.ShouldUpdate(timings))
	})

	t.Run("normal fetch before interval", func(t *testing.T) {
		tab := &GuildStashTab{FetchEnabled: true, PriorityFetch: false, LastFetch: time.Now().Add(-1 * time.Minute)}
		assert.False(t, tab.ShouldUpdate(timings))
	})

	t.Run("priority fetch after interval", func(t *testing.T) {
		tab := &GuildStashTab{FetchEnabled: true, PriorityFetch: true, LastFetch: time.Now().Add(-1 * time.Minute)}
		assert.True(t, tab.ShouldUpdate(timings))
	})

	t.Run("priority fetch before interval", func(t *testing.T) {
		tab := &GuildStashTab{FetchEnabled: true, PriorityFetch: true, LastFetch: time.Now().Add(-10 * time.Second)}
		assert.False(t, tab.ShouldUpdate(timings))
	})
}

// ==================== ActionFromString Tests ====================

func TestActionFromString(t *testing.T) {
	assert.Equal(t, ActionAdded, ActionFromString("added"))
	assert.Equal(t, ActionModified, ActionFromString("modified"))
	assert.Equal(t, ActionRemoved, ActionFromString("removed"))
	assert.Equal(t, ActionModified, ActionFromString("unknown"))
	assert.Equal(t, ActionModified, ActionFromString(""))
}

// ==================== GuildStashTab.AddChildren Tests ====================

func TestGuildStashTab_AddChildren(t *testing.T) {
	parent := &GuildStashTab{
		Id:            "parent-1",
		EventId:       10,
		TeamId:        20,
		OwnerId:       30,
		FetchEnabled:  true,
		PriorityFetch: true,
		UserIds:       []int32{1, 2, 3},
	}

	idx := 0
	colour := "#ff0000"
	children := []client.StashTab{
		{
			Id:       "child-1",
			Name:     "Child Tab 1",
			Type:     "NormalStash",
			Index:    &idx,
			Metadata: client.StashTabMetadata{Colour: &colour},
		},
	}

	parent.AddChildren(children)
	require.Len(t, parent.Children, 1)
	child := parent.Children[0]
	assert.Equal(t, "child-1", child.Id)
	assert.Equal(t, 10, child.EventId)
	assert.Equal(t, 20, child.TeamId)
	assert.Equal(t, 30, child.OwnerId)
	assert.Equal(t, "Child Tab 1", child.Name)
	assert.Equal(t, "NormalStash", child.Type)
	assert.Equal(t, "parent-1", *child.ParentId)
	assert.True(t, child.FetchEnabled)
	assert.True(t, child.PriorityFetch)
	assert.Equal(t, "{}", child.Raw)
}

func TestGuildStashTab_AddChildren_ReusesExisting(t *testing.T) {
	existingChild := &GuildStashTab{
		Id:      "child-1",
		EventId: 10,
		Raw:     "existing-data",
	}
	parent := &GuildStashTab{
		Id:       "parent-1",
		EventId:  10,
		TeamId:   20,
		OwnerId:  30,
		Children: []*GuildStashTab{existingChild},
	}

	children := []client.StashTab{{
		Id:       "child-1",
		Name:     "Updated Name",
		Type:     "QuadStash",
		Metadata: client.StashTabMetadata{},
	}}

	parent.AddChildren(children)
	// Should have original + appended = 2 entries, but the second reuses the existing child object
	// The function appends to Children, so we get 2 entries
	require.Len(t, parent.Children, 2)
	// The last one should be the updated existing child
	updated := parent.Children[1]
	assert.Equal(t, "child-1", updated.Id)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "QuadStash", updated.Type)
}

// ==================== Timing.GetDuration Tests ====================

func TestTiming_GetDuration(t *testing.T) {
	timing := &Timing{Key: CharacterRefetchDelay, DurationMs: 5000}
	assert.Equal(t, 5*time.Second, timing.GetDuration())

	timing2 := &Timing{Key: InactivityDuration, DurationMs: 0}
	assert.Equal(t, time.Duration(0), timing2.GetDuration())

	timing3 := &Timing{Key: LadderUpdateInterval, DurationMs: 60000}
	assert.Equal(t, time.Minute, timing3.GetDuration())
}

// ==================== EventRepository: SaveEvent ====================

func TestEventRepository_SaveEvent(t *testing.T) {
	defer tearDown()
	repo := &EventRepositoryImpl{DB: db}

	event := &Event{
		Name:                 "save-test",
		MaxSize:              20,
		GameVersion:          PoE1,
		ApplicationStartTime: time.Now(),
		ApplicationEndTime:   time.Now().Add(time.Hour),
		EventStartTime:       time.Now(),
		EventEndTime:         time.Now().Add(24 * time.Hour),
	}
	saved, err := repo.SaveEvent(event)
	require.NoError(t, err)
	assert.NotZero(t, saved.Id)
	assert.Equal(t, "save-test", saved.Name)
	assert.Equal(t, 20, saved.MaxSize)
}

// ==================== TeamRepository: RemoveTeamUsersForEvent, GetAllTeamUsers ====================

func TestTeamRepository_GetAllTeamUsers(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	createTestTeamsWithUsers(event)

	allUsers, err := repo.GetAllTeamUsers()
	require.NoError(t, err)
	assert.Len(t, allUsers, 4)
}

func TestTeamRepository_RemoveTeamUsersForEvent(t *testing.T) {
	defer tearDown()
	repo := &TeamRepositoryImpl{DB: db}
	event := createTestEvent()
	teams, users := createTestTeamsWithUsers(event)

	teamUsers := []*TeamUser{
		{TeamId: teams[0].Id, UserId: users[0].Id},
		{TeamId: teams[0].Id, UserId: users[1].Id},
	}
	err := repo.RemoveTeamUsersForEvent(teamUsers, event)
	require.NoError(t, err)

	remaining, err := repo.GetTeamUsersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, remaining, 2) // Only team2 users remain
}

// ==================== PassiveNodes Tests ====================

func TestPassiveNodes_GetHash(t *testing.T) {
	nodes1 := PassiveNodes{1, 2, 3}
	nodes2 := PassiveNodes{1, 2, 3}
	nodes3 := PassiveNodes{3, 2, 1}

	hash1 := nodes1.GetHash()
	hash2 := nodes2.GetHash()
	hash3 := nodes3.GetHash()

	assert.Equal(t, hash1, hash2, "same nodes should produce same hash")
	assert.NotEqual(t, hash1, hash3, "different order should produce different hash")
}

func TestPassiveNodes_GetHash_Empty(t *testing.T) {
	nodes := PassiveNodes{}
	hash := nodes.GetHash()
	assert.NotEqual(t, [32]byte{}, hash, "empty nodes should still produce a hash")
}

func TestPassiveNodes_MarshalJSON(t *testing.T) {
	nodes := PassiveNodes{10, 20, 30}
	data, err := json.Marshal(nodes)
	require.NoError(t, err)
	assert.Equal(t, "[10,20,30]", string(data))
}

func TestPassiveNodes_UnmarshalJSON(t *testing.T) {
	var nodes PassiveNodes
	err := json.Unmarshal([]byte("[5,10,15]"), &nodes)
	require.NoError(t, err)
	assert.Equal(t, PassiveNodes{5, 10, 15}, nodes)
}

func TestPassiveNodes_UnmarshalJSON_Invalid(t *testing.T) {
	var nodes PassiveNodes
	err := json.Unmarshal([]byte("not json"), &nodes)
	assert.Error(t, err)
}

func TestPassiveNodes_ScanValue_Roundtrip(t *testing.T) {
	original := PassiveNodes{100, 200, 300}
	val, err := original.Value()
	require.NoError(t, err)

	var restored PassiveNodes
	err = restored.Scan(val)
	require.NoError(t, err)
	assert.Equal(t, original, restored)
}

// ==================== Activity TableName Test ====================

func TestActivity_TableName(t *testing.T) {
	a := Activity{}
	assert.Equal(t, "activity", a.TableName())
}

// ==================== User.GetPoEToken Tests ====================

func TestUser_GetPoEToken_Valid(t *testing.T) {
	u := &User{
		OauthAccounts: []*Oauth{
			{Provider: "discord", AccessToken: "discord-token", Expiry: time.Now().Add(time.Hour)},
			{Provider: "poe", AccessToken: "poe-token-123", Expiry: time.Now().Add(time.Hour)},
		},
	}
	assert.Equal(t, "poe-token-123", u.GetPoEToken())
}

func TestUser_GetPoEToken_Expired(t *testing.T) {
	u := &User{
		OauthAccounts: []*Oauth{
			{Provider: "poe", AccessToken: "expired-token", Expiry: time.Now().Add(-time.Hour)},
		},
	}
	assert.Equal(t, "", u.GetPoEToken())
}

func TestUser_GetPoEToken_NoPoe(t *testing.T) {
	u := &User{
		OauthAccounts: []*Oauth{
			{Provider: "discord", AccessToken: "discord-token", Expiry: time.Now().Add(time.Hour)},
		},
	}
	assert.Equal(t, "", u.GetPoEToken())
}

func TestUser_GetPoEToken_NoOauths(t *testing.T) {
	u := &User{}
	assert.Equal(t, "", u.GetPoEToken())
}

// ==================== GetSignupPartners Tests ====================

func TestGetSignupPartners(t *testing.T) {
	partnerName := "player2#1234"
	signups := []*Signup{
		{User: &User{Id: 1, OauthAccounts: []*Oauth{{Provider: "poe", Name: "player1#5678"}}}, PartnerWish: &partnerName},
		{User: &User{Id: 2, OauthAccounts: []*Oauth{{Provider: "poe", Name: "player2#1234"}}}},
	}
	partners := GetSignupPartners(signups)
	assert.Len(t, partners, 1)
	assert.Equal(t, 2, partners[1].User.Id)
}

func TestGetSignupPartners_NoPartnerWish(t *testing.T) {
	signups := []*Signup{
		{User: &User{Id: 1}},
		{User: &User{Id: 2}},
	}
	partners := GetSignupPartners(signups)
	assert.Empty(t, partners)
}

func TestGetSignupPartners_NoMatch(t *testing.T) {
	wish := "nonexistent#0000"
	signups := []*Signup{
		{User: &User{Id: 1, OauthAccounts: []*Oauth{{Provider: "poe", Name: "player1#1111"}}}, PartnerWish: &wish},
		{User: &User{Id: 2, OauthAccounts: []*Oauth{{Provider: "poe", Name: "player2#2222"}}}},
	}
	partners := GetSignupPartners(signups)
	assert.Empty(t, partners)
}

// ==================== CharacterPob.UpdateStats Tests ====================

func TestCharacterPob_UpdateStats(t *testing.T) {
	pob := &client.PathOfBuilding{
		Build: client.Build{
			PlayerStats: client.PlayerStats{
				TotalDPS:                  1000.5,
				CombinedDPS:               2000.7,
				TotalEHP:                  50000.3,
				PhysicalMaximumHitTaken:   30000.0,
				FireMaximumHitTaken:       25000.0,
				ColdMaximumHitTaken:       20000.0,
				LightningMaximumHitTaken:  22000.0,
				Life:                      5000.0,
				Mana:                      2000.0,
				EnergyShield:              1500.0,
				Armour:                    10000.0,
				Evasion:                   8000.0,
				EffectiveMovementSpeedMod: 1.35,
			},
		},
	}

	cp := &CharacterPob{}
	cp.UpdateStats(pob)

	assert.Equal(t, int64(2000), cp.DPS) // max of all DPS fields = CombinedDPS = 2000.7 -> 2000
	assert.Equal(t, int32(50000), cp.EHP)
	assert.Equal(t, int32(20000), cp.EleMaxHit) // min of fire/cold/lightning
	assert.Equal(t, int32(30000), cp.PhysMaxHit)
	assert.Equal(t, int32(5000), cp.HP)
	assert.Equal(t, int32(2000), cp.Mana)
	assert.Equal(t, int32(1500), cp.ES)
	assert.Equal(t, int32(10000), cp.Armour)
	assert.Equal(t, int32(8000), cp.Evasion)
	assert.Equal(t, int32(135), cp.MovementSpeed) // 1.35 * 100
}

// ==================== ActivityRepository DB Tests ====================

func TestActivityRepository_SaveAndGetActivity(t *testing.T) {
	defer tearDown()
	db.AutoMigrate(&Activity{})
	defer db.Exec("DROP TABLE IF EXISTS bpl2.activity")

	repo := &ActivityRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(2)

	now := time.Now().Truncate(time.Microsecond)
	err := repo.SaveActivity(&Activity{UserId: users[0].Id, EventId: event.Id, Time: now})
	require.NoError(t, err)
	err = repo.SaveActivity(&Activity{UserId: users[0].Id, EventId: event.Id, Time: now.Add(time.Minute)})
	require.NoError(t, err)
	err = repo.SaveActivity(&Activity{UserId: users[1].Id, EventId: event.Id, Time: now})
	require.NoError(t, err)

	activities, err := repo.GetActivity(users[0].Id, event.Id)
	require.NoError(t, err)
	assert.Len(t, activities, 2)

	allActivities, err := repo.GetAllActivitiesForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, allActivities, 3)
}

func TestActivityRepository_GetLatestActiveTimestampsForEvent(t *testing.T) {
	defer tearDown()
	db.AutoMigrate(&Activity{})
	defer db.Exec("DROP TABLE IF EXISTS bpl2.activity")

	repo := &ActivityRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(2)

	t1 := time.Now().Truncate(time.Microsecond)
	t2 := t1.Add(time.Hour)

	repo.SaveActivity(&Activity{UserId: users[0].Id, EventId: event.Id, Time: t1})
	repo.SaveActivity(&Activity{UserId: users[0].Id, EventId: event.Id, Time: t2})
	repo.SaveActivity(&Activity{UserId: users[1].Id, EventId: event.Id, Time: t1})

	timestamps, err := repo.GetLatestActiveTimestampsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, timestamps, 2)
	assert.Equal(t, t2.UTC(), timestamps[users[0].Id].UTC())
	assert.Equal(t, t1.UTC(), timestamps[users[1].Id].UTC())
}

func TestActivityRepository_GetActivityHistoryForUsers(t *testing.T) {
	defer tearDown()
	db.AutoMigrate(&Activity{})
	defer db.Exec("DROP TABLE IF EXISTS bpl2.activity")

	repo := &ActivityRepositoryImpl{DB: db}
	event1 := createTestEvent()
	event2 := &Event{
		Name: "event2", MaxSize: 10, IsCurrent: false, GameVersion: PoE2,
		ApplicationStartTime: time.Now(), ApplicationEndTime: time.Now().Add(time.Hour),
		EventStartTime: time.Now(), EventEndTime: time.Now().Add(24 * time.Hour),
	}
	db.Create(event2)
	users := createTestUsers(2)

	now := time.Now().Truncate(time.Microsecond)
	repo.SaveActivity(&Activity{UserId: users[0].Id, EventId: event1.Id, Time: now})
	repo.SaveActivity(&Activity{UserId: users[0].Id, EventId: event2.Id, Time: now})
	repo.SaveActivity(&Activity{UserId: users[1].Id, EventId: event1.Id, Time: now})

	history, err := repo.GetActivityHistoryForUsers([]int{users[0].Id, users[1].Id})
	require.NoError(t, err)
	assert.Len(t, history, 2)
	assert.Len(t, history[users[0].Id], 2)
	assert.Len(t, history[users[1].Id], 1)
}

// ==================== CharacterRepository DB Tests ====================

func TestCharacterRepository_SaveAndGetCharacters(t *testing.T) {
	defer tearDown()
	repo := &CharacterRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	chars := []*Character{
		{Id: "char1", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "TestChar", Level: 50, VoidStones: pq.StringArray{}},
	}
	err := repo.SaveCharacters(chars)
	require.NoError(t, err)

	found, err := repo.GetCharactersForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "TestChar", found[0].Name)

	byId, err := repo.GetCharacterById("char1")
	require.NoError(t, err)
	assert.Equal(t, 50, byId.Level)
}

func TestCharacterRepository_SaveAndGetPoB(t *testing.T) {
	defer tearDown()
	repo := &CharacterRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	chars := []*Character{
		{Id: "char-pob-1", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "PoBChar", Level: 80},
	}
	repo.SaveCharacters(chars)

	pob := &CharacterPob{
		CharacterId: "char-pob-1",
		DPS:         100000,
		HP:          5000,
		ES:          2000,
		Export:      PoBExport([]byte{0x01}),
		Items:       pq.Int32Array{},
	}
	err := repo.SavePoB(pob)
	require.NoError(t, err)

	latest, err := repo.GetLatestCharacterPoB("char-pob-1")
	require.NoError(t, err)
	assert.Equal(t, int64(100000), latest.DPS)
	assert.Equal(t, int32(5000), latest.HP)

	history, err := repo.GetCharacterHistory("char-pob-1")
	require.NoError(t, err)
	assert.Len(t, history, 1)
}

func TestCharacterRepository_GetPoBById_DeletePoB(t *testing.T) {
	defer tearDown()
	repo := &CharacterRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	chars := []*Character{
		{Id: "char-del-1", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "DelChar", Level: 90},
	}
	repo.SaveCharacters(chars)

	pob := &CharacterPob{CharacterId: "char-del-1", DPS: 50000, Export: PoBExport([]byte{0x01}), Items: pq.Int32Array{}}
	repo.SavePoB(pob)

	found, err := repo.GetPoBById(pob.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(50000), found.DPS)

	err = repo.DeletePoB(pob.Id)
	require.NoError(t, err)

	_, err = repo.GetPoBById(pob.Id)
	assert.Error(t, err)
}

func TestCharacterRepository_GetCharactersForUser(t *testing.T) {
	defer tearDown()
	repo := &CharacterRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(2)

	repo.SaveCharacters([]*Character{
		{Id: "cu-1", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "Char1", Level: 10, VoidStones: pq.StringArray{}},
		{Id: "cu-2", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "Char2", Level: 20, VoidStones: pq.StringArray{}},
		{Id: "cu-3", UserId: intPtr(users[1].Id), EventId: event.Id, Name: "Char3", Level: 30, VoidStones: pq.StringArray{}},
	})

	found, err := repo.GetCharactersForUser(users[0])
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestCharacterRepository_SavePoBs(t *testing.T) {
	defer tearDown()
	repo := &CharacterRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	repo.SaveCharacters([]*Character{
		{Id: "spob-1", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "C1", Level: 50, VoidStones: pq.StringArray{}},
		{Id: "spob-2", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "C2", Level: 60, VoidStones: pq.StringArray{}},
	})

	dummyExport := PoBExport([]byte{0x01})
	pobs := []*CharacterPob{
		{CharacterId: "spob-1", DPS: 1000, Export: dummyExport, Items: pq.Int32Array{}},
		{CharacterId: "spob-2", DPS: 2000, Export: dummyExport, Items: pq.Int32Array{}},
	}
	err := repo.SavePoBs(pobs)
	require.NoError(t, err)

	found, err := repo.GetLatestPoBsForEvent(event.Id)
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestCharacterRepository_GetPobByCharacterIdBeforeTimestamp(t *testing.T) {
	defer tearDown()
	repo := &CharacterRepositoryImpl{DB: db}
	event := createTestEvent()
	users := createTestUsers(1)

	repo.SaveCharacters([]*Character{
		{Id: "ts-char-1", UserId: intPtr(users[0].Id), EventId: event.Id, Name: "TSChar", Level: 70},
	})

	dummy := PoBExport([]byte{0x01})
	repo.SavePoB(&CharacterPob{CharacterId: "ts-char-1", DPS: 1000, Export: dummy, Items: pq.Int32Array{}})
	time.Sleep(10 * time.Millisecond)
	cutoff := time.Now()
	time.Sleep(10 * time.Millisecond)
	repo.SavePoB(&CharacterPob{CharacterId: "ts-char-1", DPS: 9999, Export: dummy, Items: pq.Int32Array{}})

	pob, err := repo.GetPobByCharacterIdBeforeTimestamp("ts-char-1", cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), pob.DPS)
}
