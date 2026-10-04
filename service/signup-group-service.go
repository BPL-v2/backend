package service

import "bpl/repository"

type SignupGroupService interface {
	CreateGroup(eventId int, userId int, maxGroupSize int) (string, error)
	JoinGroup(eventId int, userId int, groupKey string, maxGroupSize int) error
	LeaveGroup(eventId int, userId int) error
	LockGroupsOfUsers(eventId int, userIds []int) error
	GetGroupKeyForUser(eventId int, userId int) (*string, error)
	GetGroupMembers(eventId int, groupKey string) ([]*repository.SignupGroupMember, error)
	GetGroupKeysForEvent(eventId int) (map[int]string, error)
}

type SignupGroupServiceImpl struct {
	repo repository.SignupGroupRepository
}

func NewSignupGroupService() SignupGroupService {
	return &SignupGroupServiceImpl{repo: repository.NewSignupGroupRepository()}
}

func (s *SignupGroupServiceImpl) CreateGroup(eventId int, userId int, maxGroupSize int) (string, error) {
	return s.repo.CreateGroup(eventId, userId, maxGroupSize)
}

func (s *SignupGroupServiceImpl) JoinGroup(eventId int, userId int, groupKey string, maxGroupSize int) error {
	return s.repo.JoinGroup(eventId, userId, groupKey, maxGroupSize)
}

func (s *SignupGroupServiceImpl) LeaveGroup(eventId int, userId int) error {
	return s.repo.LeaveGroup(eventId, userId)
}

func (s *SignupGroupServiceImpl) GetGroupKeyForUser(eventId int, userId int) (*string, error) {
	return s.repo.GetGroupKeyForUser(eventId, userId)
}

func (s *SignupGroupServiceImpl) GetGroupMembers(eventId int, groupKey string) ([]*repository.SignupGroupMember, error) {
	return s.repo.GetGroupMembers(eventId, groupKey)
}

func (s *SignupGroupServiceImpl) GetGroupKeysForEvent(eventId int) (map[int]string, error) {
	return s.repo.GetGroupKeysForEvent(eventId)
}

func (s *SignupGroupServiceImpl) LockGroupsOfUsers(eventId int, userIds []int) error {
	return s.repo.LockGroupsOfUsers(eventId, userIds)
}
