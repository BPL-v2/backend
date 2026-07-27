package service

import (
	"bpl/client"
	"bpl/repository"
	"bpl/utils"
	"math"
)

type ItemWishService interface {
	CreateItemWish(itemWish *repository.ItemWish, teamId int) (*repository.ItemWish, error)
	UpdateItemWish(itemWish *repository.ItemWish, teamId int, Fulfilled *bool, BuildEnabling *bool, Priority *int) (*repository.ItemWish, error)
	GetItemWishById(id int) (*repository.ItemWish, error)
	DeleteItemWish(id int) error
	GetItemWishesForTeam(teamId int) ([]*repository.ItemWish, error)
	UpdateItemWishFulfillment(teamId int, userId int, character *client.Character) error
}

type ItemWishServiceImpl struct {
	itemWishRepository repository.ItemWishRepository
}

func NewItemWishService() ItemWishService {
	return &ItemWishServiceImpl{
		itemWishRepository: repository.NewItemWishRepository(),
	}
}

func (s *ItemWishServiceImpl) CreateItemWish(itemWish *repository.ItemWish, teamId int) (*repository.ItemWish, error) {
	itemWishes, err := s.itemWishRepository.GetSimilarItemWishesInTeam(teamId, itemWish.ItemField, itemWish.Value)
	if err != nil {
		return nil, err
	}
	itemWish.Priority = len(itemWishes)
	return s.itemWishRepository.SaveItemWish(itemWish)
}

func (s *ItemWishServiceImpl) UpdateItemWish(itemWish *repository.ItemWish, teamId int, Fulfilled *bool, BuildEnabling *bool, Priority *int) (*repository.ItemWish, error) {
	if Fulfilled != nil {
		itemWish.Fulfilled = *Fulfilled
	}
	if BuildEnabling != nil {
		itemWish.BuildEnabling = *BuildEnabling
	}
	if Priority != nil {
		itemWishes, err := s.itemWishRepository.GetSimilarItemWishesInTeam(teamId, itemWish.ItemField, itemWish.Value)
		if err != nil {
			return nil, err
		}
		priority := int(math.Max(math.Min(float64(*Priority), float64(len(itemWishes)-1)), 0))
		for _, iw := range itemWishes {
			if iw.Priority == priority {
				iw.Priority = itemWish.Priority
				_, err = s.itemWishRepository.SaveItemWish(iw)
				if err != nil {
					return nil, err
				}
				break
			}
		}
		itemWish.Priority = priority
	}
	return s.itemWishRepository.SaveItemWish(itemWish)
}

func (s *ItemWishServiceImpl) GetItemWishById(id int) (*repository.ItemWish, error) {
	return s.itemWishRepository.GetItemWishById(id)
}

func (s *ItemWishServiceImpl) DeleteItemWish(id int) error {
	return s.itemWishRepository.DeleteItemWish(id)
}

func (s *ItemWishServiceImpl) GetItemWishesForTeam(teamId int) ([]*repository.ItemWish, error) {
	itemWishes, err := s.itemWishRepository.GetItemWishesForTeam(teamId)
	if err != nil {
		return nil, err
	}
	return itemWishes, nil
}

func (s *ItemWishServiceImpl) UpdateItemWishFulfillment(teamId int, userId int, character *client.Character) error {
	itemWishes, err := s.itemWishRepository.GetItemWishesForTeamAndUser(teamId, userId)
	if err != nil {
		return err
	}
	toSave := make([]*repository.ItemWish, 0)
	for _, itemWish := range itemWishes {
		if itemWish.Fulfilled || character.Equipment == nil {
			continue
		}
		for _, item := range utils.FlatMap(*character.Equipment, func(i client.Item) []client.Item {
			items := []client.Item{i}
			if i.SocketedItems != nil {
				items = append(items, *i.SocketedItems...)
			}
			return items
		}) {
			switch itemWish.ItemField {
			case repository.BASE_TYPE:
				if item.BaseType == itemWish.Value {
					itemWish.Fulfilled = true
					toSave = append(toSave, itemWish)
				}
			case repository.NAME:
				if item.Name == itemWish.Value {
					itemWish.Fulfilled = true
					toSave = append(toSave, itemWish)
				}
			default:
				continue
			}
		}
	}
	if len(toSave) > 0 {
		_, err = s.itemWishRepository.SaveItemWishes(toSave)
		if err != nil {
			return err
		}
	}
	return nil
}
