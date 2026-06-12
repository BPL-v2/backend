package repository

import (
	"bpl/config"
	"bpl/metrics"
	"bpl/utils"
	"database/sql/driver"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Permission string

const (
	PermissionAdmin              Permission = "admin"
	PermissionManager            Permission = "manager"
	PermissionObjectiveDesigner  Permission = "objective_designer"
	PermissionSubmissionJudge    Permission = "submission_judge"
	PermissionAchievementAssigner Permission = "achievement_assigner"
)

type Permissions []Permission

func (p *Permissions) Scan(src any) error {
	x := make(pq.StringArray, 0)
	err := x.Scan(src)
	if err != nil {
		return err
	}
	permissions := make(Permissions, len(x))
	for i, perm := range x {
		permissions[i] = Permission(perm)
	}
	*p = permissions
	return nil
}

func (p Permissions) Value() (driver.Value, error) {
	permissions := make(pq.StringArray, len(p))
	for i, perm := range p {
		permissions[i] = string(perm)
	}
	return permissions.Value()
}

type User struct {
	Id          int         `gorm:"primaryKey autoIncrement"`
	DisplayName string      `gorm:"not null"`
	Permissions Permissions `gorm:"type:text[];not null;default:'{}'"`
	Referrer    *string     `gorm:"type:text;null"`

	OauthAccounts []*Oauth `gorm:"foreignKey:UserId"`
}

func (u *User) HasOneOfPermissions(perms ...Permission) bool {
	for _, perm := range perms {
		if slices.Contains(u.Permissions, perm) {
			return true
		}
	}
	return false
}

func (u *User) GetPoEToken() string {
	for _, oauth := range u.OauthAccounts {
		if oauth.Provider == "poe" {
			if oauth.Expiry.Before(time.Now()) {
				return ""
			}
			return oauth.AccessToken
		}
	}
	return ""
}

func (u *User) GetAccountName(provider Provider) *string {
	for _, oauth := range u.OauthAccounts {
		if oauth.Provider == provider {
			return &oauth.Name
		}
	}
	return nil
}

func (u *User) HasPoEName(name string) bool {
	poeName := u.GetAccountName(ProviderPoE)
	if poeName == nil {
		return false
	}
	return strings.EqualFold(strings.Split(*poeName, "#")[0], strings.Split(name, "#")[0])
}

type UserRepository interface {
	GetUserById(userId int, preloads ...string) (*User, error)
	GetUsersByIds(userIds []int, preloads ...string) ([]*User, error)
	SaveUser(user *User) (*User, error)
	GetAllUsers() ([]*User, error)
	GetStreamersForEvent(eventId int) (streamers []*Streamer, err error)
	GetUsersForEvent(eventId int) ([]*TeamUserWithPoEToken, error)
	GetUsersWithTeamForEvent(eventId int) (map[int]*UserWithTeam, error)
}

type UserRepositoryImpl struct {
	DB *gorm.DB
}

func NewUserRepository() UserRepository {
	return &UserRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *UserRepositoryImpl) GetUserById(userId int, preloads ...string) (*User, error) {
	var user User
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.First(&user, userId)
	if result.Error != nil {
		return nil, fmt.Errorf("user with id %d not found", userId)
	}
	return &user, nil
}
func (r *UserRepositoryImpl) GetUsersByIds(userIds []int, preloads ...string) ([]*User, error) {
	var users []*User
	query := r.DB
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	result := query.Find(&users, userIds)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users by ids: %v", result.Error)
	}
	return users, nil
}

func (r *UserRepositoryImpl) SaveUser(user *User) (*User, error) {
	result := r.DB.Save(user)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to create user: %v", result.Error)
	}
	return user, nil
}

func (r *UserRepositoryImpl) GetAllUsers() ([]*User, error) {
	var users []*User
	result := r.DB.Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users: %v", result.Error)
	}
	return users, nil
}

type Streamer struct {
	UserId   int
	TwitchId string
}

func (r *UserRepositoryImpl) GetStreamersForEvent(eventId int) (streamers []*Streamer, err error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetStreamersForCurrentEvent"))
	defer timer.ObserveDuration()
	query := `
		SELECT 
			users.id as user_id, 
			oauths.account_id as twitch_id
		FROM users
		JOIN oauths ON oauths.user_id = users.id
		JOIN team_users ON team_users.user_id = users.id
		JOIN teams ON teams.id = team_users.team_id
		WHERE teams.event_id = ? AND oauths.provider = 'twitch'
	`
	result := r.DB.Raw(query, eventId).Scan(&streamers)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get streamers for current event: %v", result.Error)
	}
	return streamers, nil

}

type UserTeam struct {
	UserId      int
	TeamId      int
	DisplayName string
}

func LoadUsersIntoEvent(DB *gorm.DB, event *Event) error {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("LoadUsersIntoEvent"))
	defer timer.ObserveDuration()
	var users []*UserTeam
	query := `
		SELECT
			users.id as user_id,
			users.display_name as display_name,	
			team_users.team_id as team_id
		FROM users
		JOIN team_users ON team_users.user_id = users.id
		WHERE team_users.team_id IN ?		
		`
	err := DB.Raw(query, utils.Map(event.Teams, func(team *Team) int {
		return team.Id
	})).Scan(&users).Error
	if err != nil {
		log.Print(err)
		return fmt.Errorf("failed to load users into event: %v", err)
	}
	for _, user := range users {
		for _, team := range event.Teams {
			if team.Id == user.TeamId {
				team.Users = append(team.Users, &User{Id: user.UserId, DisplayName: user.DisplayName})
			}
		}
	}
	return nil

}

type TeamUserWithPoEToken struct {
	TeamId      int
	UserId      int
	AccountName string
	Token       string
	TokenExpiry time.Time
}

func (r *UserRepositoryImpl) GetUsersForEvent(eventId int) ([]*TeamUserWithPoEToken, error) {
	timer := prometheus.NewTimer(metrics.QueryDuration.WithLabelValues("GetUsersForEvent"))
	defer timer.ObserveDuration()
	var users []*TeamUserWithPoEToken
	query := `
		SELECT
			users.id as user_id,
			oauths.name as account_name,
			oauths.access_token as token,
			oauths.expiry as token_expiry,
			team_users.team_id as team_id
		FROM users
		JOIN oauths ON oauths.user_id = users.id
		JOIN team_users ON team_users.user_id = users.id
		JOIN teams ON teams.id = team_users.team_id
		WHERE teams.event_id = ? AND oauths.provider = 'poe'
	`
	result := r.DB.Raw(query, eventId).Scan(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users for event: %v", result.Error)
	}
	return users, nil
}

type UserWithTeam struct {
	UserId      int
	PoEAccount  string
	DiscordName string
	DiscordId   string
	TwitchName  string
	TeamId      int
}

func (r *UserRepositoryImpl) GetUsersWithTeamForEvent(eventId int) (map[int]*UserWithTeam, error) {
	query := `
		SELECT
			users.id as user_id,
			oauths.provider as provider,
			oauths.name as account_name,
			oauths.account_id as account_id,
			team_users.team_id as team_id
		FROM users
	JOIN oauths ON oauths.user_id = users.id
	JOIN team_users ON team_users.user_id = users.id
	JOIN teams ON teams.id = team_users.team_id
	WHERE teams.event_id = ?
	`
	type Result struct {
		UserId      int
		Provider    Provider
		AccountName string
		AccountId   string
		TeamId      int
	}
	var users []*Result
	result := r.DB.Raw(query, eventId).Scan(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get users with team for event: %v", result.Error)
	}
	mapped := make(map[int]*UserWithTeam)
	for _, user := range users {
		if _, ok := mapped[user.UserId]; !ok {
			mapped[user.UserId] = &UserWithTeam{UserId: user.UserId, TeamId: user.TeamId}
		}
		switch user.Provider {
		case ProviderPoE:
			mapped[user.UserId].PoEAccount = user.AccountName
		case ProviderDiscord:
			mapped[user.UserId].DiscordName = user.AccountName
			mapped[user.UserId].DiscordId = user.AccountId
		case ProviderTwitch:
			mapped[user.UserId].TwitchName = user.AccountName
		}
	}
	return mapped, nil
}
