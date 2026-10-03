package repository

import (
	"bpl/config"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrGroupNotFound  = errors.New("group does not exist")
	ErrGroupFull      = errors.New("group is full")
	ErrGroupsDisabled = errors.New("group signups are not enabled for this event")
	ErrGroupLocked    = errors.New("group is locked because players were already sorted into teams")
	ErrAlreadyInGroup = errors.New("already in a group")
	ErrNotInGroup     = errors.New("not in a group")
)

type SignupGroupMember struct {
	EventId  int       `gorm:"not null;primaryKey"`
	UserId   int       `gorm:"not null;primaryKey"`
	GroupKey string    `gorm:"not null;index"`
	Locked   bool      `gorm:"not null;default:false"`
	JoinedAt time.Time `gorm:"not null;autoCreateTime;default:CURRENT_TIMESTAMP"`
	User     *User     `gorm:"foreignKey:UserId;references:Id"`
}

type SignupGroupRepository interface {
	CreateGroup(eventId int, userId int, maxGroupSize int) (string, error)
	JoinGroup(eventId int, userId int, groupKey string, maxGroupSize int) error
	LeaveGroup(eventId int, userId int) error
	LockGroupsOfUsers(eventId int, userIds []int) error
	GetGroupKeyForUser(eventId int, userId int) (*string, error)
	GetGroupMembers(eventId int, groupKey string) ([]*SignupGroupMember, error)
	GetGroupKeysForEvent(eventId int) (map[int]string, error)
}

type SignupGroupRepositoryImpl struct {
	DB *gorm.DB
}

func NewSignupGroupRepository() SignupGroupRepository {
	return &SignupGroupRepositoryImpl{DB: config.DatabaseConnection()}
}

func (r *SignupGroupRepositoryImpl) CreateGroup(eventId int, userId int, maxGroupSize int) (string, error) {
	key := uuid.NewString()
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&SignupGroupMember{}).Where("event_id = ? AND user_id = ?", eventId, userId).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return ErrAlreadyInGroup
		}
		return tx.Create(&SignupGroupMember{EventId: eventId, UserId: userId, GroupKey: key}).Error
	})
	return key, err
}

func (r *SignupGroupRepositoryImpl) JoinGroup(eventId int, userId int, groupKey string, maxGroupSize int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&SignupGroupMember{}).Where("event_id = ? AND user_id = ?", eventId, userId).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return ErrAlreadyInGroup
		}
		// lock the existing rows of the group so concurrent joins are serialized,
		// the count afterwards is a new statement and sees rows committed in the meantime
		locked := make([]*SignupGroupMember, 0)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("event_id = ? AND group_key = ?", eventId, groupKey).Find(&locked).Error; err != nil {
			return err
		}
		if len(locked) == 0 {
			return ErrGroupNotFound
		}
		var size int64
		if err := tx.Model(&SignupGroupMember{}).Where("event_id = ? AND group_key = ?", eventId, groupKey).Count(&size).Error; err != nil {
			return err
		}
		if int(size) >= maxGroupSize {
			return ErrGroupFull
		}
		if locked[0].Locked {
			return ErrGroupLocked
		}
		return tx.Create(&SignupGroupMember{EventId: eventId, UserId: userId, GroupKey: groupKey}).Error
	})
}

func (r *SignupGroupRepositoryImpl) LeaveGroup(eventId int, userId int) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		member := SignupGroupMember{}
		if err := tx.First(&member, "event_id = ? AND user_id = ?", eventId, userId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotInGroup
			}
			return err
		}
		if member.Locked {
			return ErrGroupLocked
		}
		return tx.Delete(&SignupGroupMember{}, "event_id = ? AND user_id = ?", eventId, userId).Error
	})
}

// LockGroupsOfUsers locks every group that one of the users is a member of.
func (r *SignupGroupRepositoryImpl) LockGroupsOfUsers(eventId int, userIds []int) error {
	if len(userIds) == 0 {
		return nil
	}
	keys := r.DB.Model(&SignupGroupMember{}).Select("group_key").Where("event_id = ? AND user_id IN ?", eventId, userIds)
	return r.DB.Model(&SignupGroupMember{}).Where("event_id = ? AND group_key IN (?)", eventId, keys).Update("locked", true).Error
}

func (r *SignupGroupRepositoryImpl) GetGroupKeyForUser(eventId int, userId int) (*string, error) {
	member := SignupGroupMember{}
	err := r.DB.First(&member, "event_id = ? AND user_id = ?", eventId, userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &member.GroupKey, nil
}

func (r *SignupGroupRepositoryImpl) GetGroupMembers(eventId int, groupKey string) ([]*SignupGroupMember, error) {
	members := make([]*SignupGroupMember, 0)
	err := r.DB.Preload("User").Preload("User.OauthAccounts").
		Where("event_id = ? AND group_key = ?", eventId, groupKey).
		Order("joined_at ASC").Find(&members).Error
	return members, err
}

func (r *SignupGroupRepositoryImpl) GetGroupKeysForEvent(eventId int) (map[int]string, error) {
	members := make([]*SignupGroupMember, 0)
	if err := r.DB.Where("event_id = ?", eventId).Find(&members).Error; err != nil {
		return nil, err
	}
	keys := make(map[int]string, len(members))
	for _, m := range members {
		keys[m.UserId] = m.GroupKey
	}
	return keys, nil
}
