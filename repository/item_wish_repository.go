package repository

import (
	"bpl/config"

	"gorm.io/gorm"
)

type ItemWishRepository interface {
	SaveItemWish(itemWish *ItemWish) (*ItemWish, error)
	SaveItemWishes(itemWishes []*ItemWish) ([]*ItemWish, error)
	GetItemWishesForTeamAndUser(teamId int, userId int) (itemWishes []*ItemWish, err error)
	GetItemWishesForTeam(teamId int) (itemWishes []*ItemWish, err error)
	GetSimilarItemWishesInTeam(teamId int, itemField ItemField, value string) (itemWishes []*ItemWish, err error)
	DeleteItemWish(id int) error
	GetItemWishById(id int) (*ItemWish, error)
}

type ItemWishRepositoryImpl struct {
	DB *gorm.DB
}

func NewItemWishRepository() ItemWishRepository {
	return &ItemWishRepositoryImpl{DB: config.DatabaseConnection()}
}

type ItemWish struct {
	Id        int       `gorm:"not null;primaryKey;autoIncrement"`
	UserID    int       `gorm:"not null;index:idx_user_event_item_wish"`
	TeamID    int       `gorm:"not null;index:idx_user_team_item_wish"`
	ItemField ItemField `gorm:"not null"`
	Value     string    `gorm:"not null"`
	Extra     string    `gorm:"not null;default:''"`
	Fulfilled bool      `gorm:"not null;default:false"`
	// BuildEnabling is a 1-5 importance scale (1 = nice to have, 5 = absolutely
	// essential). Legacy boolean values map false -> 1 and true -> 5.
	BuildEnabling int `gorm:"not null;default:1"`
	Priority      int `gorm:"not null;default:0"`

	User *User `gorm:"foreignKey:UserID"`
	Team *Team `gorm:"foreignKey:TeamID"`
}

func (r *ItemWishRepositoryImpl) SaveItemWish(itemWish *ItemWish) (*ItemWish, error) {
	err := r.DB.Save(itemWish).Error
	return itemWish, err
}
func (r *ItemWishRepositoryImpl) SaveItemWishes(itemWishes []*ItemWish) ([]*ItemWish, error) {
	err := r.DB.Save(itemWishes).Error
	return itemWishes, err
}

func (r *ItemWishRepositoryImpl) GetItemWishesForTeamAndUser(teamId int, userId int) (itemWishes []*ItemWish, err error) {
	err = r.DB.Where("team_id = ? AND user_id = ?", teamId, userId).Find(&itemWishes).Error
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (r *ItemWishRepositoryImpl) GetItemWishesForTeam(teamId int) (itemWishes []*ItemWish, err error) {
	err = r.DB.Where("team_id = ?", teamId).Find(&itemWishes).Error
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (r *ItemWishRepositoryImpl) GetSimilarItemWishesInTeam(teamId int, itemField ItemField, value string) (itemWishes []*ItemWish, err error) {
	err = r.DB.Where("team_id = ? AND item_field = ? AND value = ?", teamId, itemField, value).Find(&itemWishes).Error
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (r *ItemWishRepositoryImpl) DeleteItemWish(id int) error {
	return r.DB.Delete(&ItemWish{}, id).Error
}

func (r *ItemWishRepositoryImpl) GetItemWishById(id int) (*ItemWish, error) {
	var itemWish ItemWish
	err := r.DB.First(&itemWish, id).Error
	if err != nil {
		return nil, err
	}
	return &itemWish, nil
}
