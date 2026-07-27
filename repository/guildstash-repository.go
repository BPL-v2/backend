package repository

import (
	"bpl/client"
	"bpl/config"
	"strings"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type GuildStashTab struct {
	Id            string        `gorm:"primaryKey;not null"`
	EventId       int           `gorm:"primaryKey;not null;index:guild_stash_tab_event_idx"`
	TeamId        int           `gorm:"not null;index:guild_stash_tab_team_idx"`
	OwnerId       int           `gorm:"not null"`
	Name          string        `gorm:"not null"`
	Type          string        `gorm:"not null"`
	Index         *int          `gorm:"null"`
	Color         *string       `gorm:"null"`
	ParentId      *string       `gorm:"null;references:Id"`
	ParentEventId *int          `gorm:"null"`
	Raw           string        `gorm:"type:text;not null;default:''"`
	FetchEnabled  bool          `gorm:"not null"`
	PriorityFetch bool          `gorm:"not null;default:false"`
	UserIds       pq.Int32Array `gorm:"not null;type:integer[]"`
	LastFetch     time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP"`

	Event    Event            `gorm:"foreignKey:EventId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Team     Team             `gorm:"foreignKey:TeamId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Parent   *GuildStashTab   `gorm:"foreignKey:ParentId,ParentEventId;references:Id,EventId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Owner    User             `gorm:"foreignKey:OwnerId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Children []*GuildStashTab `gorm:"foreignKey:ParentId,ParentEventId;references:Id,EventId;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (g *GuildStashTab) ShouldUpdate(timings map[TimingKey]time.Duration) bool {
	if !g.FetchEnabled {
		return false
	}
	if g.PriorityFetch {
		return time.Since(g.LastFetch) > timings[GuildstashPriorityFetchInterval]
	}
	return time.Since(g.LastFetch) > timings[GuildstashUpdateInterval]
}

func (g *GuildStashTab) AddChildren(children []client.StashTab) {
	existingChildren := make(map[string]*GuildStashTab)
	for _, child := range g.Children {
		existingChildren[child.Id] = child
	}
	for _, child := range children {
		c := &GuildStashTab{}
		if existingChild, exists := existingChildren[child.Id]; exists {
			c = existingChild
		}
		c.Id = child.Id
		c.EventId = g.EventId
		c.TeamId = g.TeamId
		c.OwnerId = g.OwnerId
		c.Name = child.Name
		c.Type = child.Type
		c.Index = child.Index
		c.Color = child.Metadata.Colour
		c.ParentId = &g.Id
		c.ParentEventId = &g.EventId
		c.LastFetch = time.Now()
		c.Raw = "{}"
		c.FetchEnabled = g.FetchEnabled
		c.PriorityFetch = g.PriorityFetch
		c.UserIds = g.UserIds
		g.Children = append(g.Children, c)

	}

}

type GuildStashChangelog struct {
	Id          int       `gorm:"primaryKey"`
	Timestamp   time.Time `gorm:"not null"`
	GuildId     int       `gorm:"not null"`
	EventId     int       `gorm:"not null"`
	StashName   *string   `gorm:"null"`
	AccountName string    `gorm:"not null"`
	Action      Action    `gorm:"not null"`
	Number      int       `gorm:"not null"`
	ItemName    string    `gorm:"not null"`
	X           int       `gorm:"not null"`
	Y           int       `gorm:"not null"`
}

type Guild struct {
	Id      int `gorm:"primaryKey"`
	TeamId  int `gorm:"primaryKey"`
	EventId int `gorm:"primaryKey"`
	Name    string
	Tag     string
}

type Action int

const (
	ActionAdded    Action = 1
	ActionModified Action = 0
	ActionRemoved  Action = -1
)

func ActionFromString(action string) Action {
	switch action {
	case "added":
		return ActionAdded
	case "modified":
		return ActionModified
	case "removed":
		return ActionRemoved
	default:
		return ActionModified
	}
}

type GuildStashRepository interface {
	DeleteAll(tabs []*GuildStashTab) error
	Delete(stashId string, eventId int) error
	SaveAll(tabs []*GuildStashTab) (err error)
	Save(tab *GuildStashTab) error
	GetById(stashId string, eventId int) (tab *GuildStashTab, err error)
	GetByEvent(eventId int) ([]*GuildStashTab, error)
	GetActiveByEvent(eventId int) ([]*GuildStashTab, error)
	GetByTeam(teamId int) ([]*GuildStashTab, error)
	GetByUserAndEvent(userId int, eventId int) ([]*GuildStashTab, error)
	SwitchStashFetch(stashId string, teamId int, fetchEnabled bool, priorityFetch bool) error
	SaveGuildstashLogs(logs []*GuildStashChangelog) error
	GetLatestLogEntryTimestampForGuild(event *Event, guildId int) (*int64, *int64)
	GetLogs(eventId, guildId int, limit, offset *int, userName, stashName, itemName *string) ([]*GuildStashChangelog, error)
	SaveGuild(guild *Guild) error
	GetGuildsForTeams(teamIds []int) ([]*Guild, error)
	GetEarliestDeposits(event *Event) ([]*PlayerCompletion, error)
	GetGuildById(guildId int, eventId int) (*Guild, error)
}

type GuildStashRepositoryImpl struct {
	db *gorm.DB
}

func NewGuildStashRepository() GuildStashRepository {
	return &GuildStashRepositoryImpl{
		db: config.DatabaseConnection(),
	}
}

func (r *GuildStashRepositoryImpl) DeleteAll(tabs []*GuildStashTab) error {
	if len(tabs) == 0 {
		return nil
	}
	return r.db.Delete(tabs).Error
}

func (r *GuildStashRepositoryImpl) Delete(stashId string, eventId int) error {
	return r.db.Where("(id = ? OR parent_id = ?) AND event_id = ?", stashId, stashId, eventId).Delete(&GuildStashTab{}).Error
}

func (r *GuildStashRepositoryImpl) SaveAll(tabs []*GuildStashTab) (err error) {
	if len(tabs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, tab := range tabs {
			err = r.db.Save(tab).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GuildStashRepositoryImpl) Save(tab *GuildStashTab) error {
	return r.db.Save(tab).Error
}
func (r *GuildStashRepositoryImpl) GetById(stashId string, eventId int) (tab *GuildStashTab, err error) {
	allTabs := make([]*GuildStashTab, 0)
	q := `select * from guild_stash_tabs where (id = @stash_id OR parent_id = @stash_id) and event_id = @event_id`
	err = r.db.Raw(q, map[string]any{
		"stash_id": stashId,
		"event_id": eventId,
	}).Scan(&allTabs).Error
	if err != nil {
		return nil, err
	}
	children := make([]*GuildStashTab, 0)
	for _, t := range allTabs {
		if t.Id == stashId {
			tab = t
		} else {
			children = append(children, t)
		}
	}
	if tab == nil {
		return nil, gorm.ErrRecordNotFound
	}
	tab.Children = children
	return tab, err
}

func (r *GuildStashRepositoryImpl) GetByEvent(eventId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{EventId: eventId}).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepositoryImpl) GetActiveByEvent(eventId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{EventId: eventId, FetchEnabled: true}).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepositoryImpl) GetByTeam(teamId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where(GuildStashTab{TeamId: teamId}).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepositoryImpl) GetByUserAndEvent(userId int, eventId int) ([]*GuildStashTab, error) {
	var tabs []*GuildStashTab
	err := r.db.Where("event_id = ? AND ? = ANY(user_ids)", eventId, userId).Find(&tabs).Error
	if err != nil {
		return nil, err
	}
	return tabs, nil
}

func (r *GuildStashRepositoryImpl) SwitchStashFetch(stashId string, teamId int, fetchEnabled bool, priorityFetch bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		query := `
		UPDATE guild_stash_tabs g
		SET fetch_enabled = @fetch_enabled, priority_fetch = @priority_fetch
		WHERE (g.id = @stash_id OR g.parent_id = @stash_id) AND g.team_id = @team_id;
	`
		err := r.db.Exec(query, map[string]any{
			"fetch_enabled":  fetchEnabled,
			"priority_fetch": priorityFetch,
			"stash_id":       stashId,
			"team_id":        teamId,
		}).Error
		if err != nil {
			return err
		}
		if priorityFetch {
			query = `
				UPDATE guild_stash_tabs g
				SET priority_fetch = false
				WHERE (g.id != @stash_id OR g.parent_id != @stash_id) AND g.team_id = @team_id;
			`
			err = r.db.Exec(query, map[string]any{
				"stash_id": stashId,
				"team_id":  teamId,
			}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GuildStashRepositoryImpl) SaveGuildstashLogs(logs []*GuildStashChangelog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.Save(logs).Error
}

func (r *GuildStashRepositoryImpl) GetLatestLogEntryTimestampForGuild(event *Event, guildId int) (*int64, *int64) {
	var result struct {
		EarliestTimestamp *time.Time
		LatestTimestamp   *time.Time
	}
	err := r.db.Model(&GuildStashChangelog{}).
		Select("MIN(timestamp) as earliest_timestamp, MAX(timestamp) as latest_timestamp").
		Where("event_id = ? AND guild_id = ?", event.Id, guildId).
		Scan(&result).Error

	if err != nil || result.EarliestTimestamp == nil || result.LatestTimestamp == nil {
		return nil, nil
	}
	earliestTimestamp := result.EarliestTimestamp.Unix() - 1
	latestTimestamp := result.LatestTimestamp.Unix() + 1
	return &earliestTimestamp, &latestTimestamp
}

func (r *GuildStashRepositoryImpl) GetLogs(eventId, guildId int, limit, offset *int, userName, stashName, itemName *string) ([]*GuildStashChangelog, error) {
	var logs []*GuildStashChangelog
	query := r.db.Model(&GuildStashChangelog{})
	query = query.Where("event_id = ? AND guild_id = ?", eventId, guildId)
	if userName != nil {
		query = query.Where("account_name = ?", strings.ReplaceAll(*userName, "-", "#"))
	}
	if stashName != nil {
		query = query.Where("stash_name = ?", *stashName)
	}
	if itemName != nil {
		query = query.Where("item_name ILIKE ?", "%"+*itemName+"%")
	}
	if limit != nil {
		query = query.Limit(*limit)
	}
	if offset != nil {
		query = query.Offset(*offset)
	}
	err := query.Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *GuildStashRepositoryImpl) SaveGuild(guild *Guild) error {
	return r.db.Save(guild).Error
}

func (r *GuildStashRepositoryImpl) GetGuildsForTeams(teamIds []int) ([]*Guild, error) {
	var guilds []*Guild
	err := r.db.Where("team_id IN ?", teamIds).Find(&guilds).Error
	if err != nil {
		return nil, err
	}
	return guilds, nil
}

type PlayerCompletion struct {
	Timestamp time.Time
	UserId    int    `gorm:"column:user_id"`
	ItemName  string `gorm:"column:item_name"`
	TeamId    int    `gorm:"column:team_id"`
}

func (r *GuildStashRepositoryImpl) GetEarliestDeposits(event *Event) ([]*PlayerCompletion, error) {
	var results []*PlayerCompletion
	query := `
	SELECT timestamp, user_id, item_name, team_id 
		FROM (
			SELECT 
				gsc.timestamp, 
				o.user_id, 
				gsc.item_name, 
				g.team_id,
				ROW_NUMBER() OVER (
					PARTITION BY gsc.item_name, g.team_id 
					ORDER BY gsc.timestamp ASC
				) as rn
			FROM guild_stash_changelogs gsc 
			JOIN guilds g ON g.id = gsc.guild_id 
			JOIN oauths o ON o."name" = gsc.account_name
			WHERE gsc.action = 1 AND gsc.number = 1 AND g.team_id IN ? AND gsc.timestamp >= ? AND gsc.timestamp <= ?
		) ranked
		WHERE rn = 1
		ORDER BY timestamp;
		`
	err := r.db.Raw(query, event.TeamIds(), event.EventStartTime, event.EventEndTime).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (r *GuildStashRepositoryImpl) GetGuildById(guildId int, eventId int) (*Guild, error) {
	var guild Guild
	err := r.db.Where("id = ? AND event_id = ?", guildId, eventId).First(&guild).Error
	if err != nil {
		return nil, err
	}
	return &guild, nil
}
