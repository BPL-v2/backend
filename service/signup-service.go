package service

import (
	"bpl/repository"
	"time"
)

type SignupService interface {
	SaveSignup(signup *repository.Signup) (*repository.Signup, error)
	RemoveSignupForUser(userId int, eventId int) error
	GetSignupForUser(userId int, eventId int) (*repository.Signup, error)
	GetSignupsForEvent(event *repository.Event) ([]*repository.Signup, error)
	GetExtendedSignupsForEvent(event *repository.Event) ([]*repository.Signup, map[int]map[int]time.Duration, map[int]map[int]int, error)
}

type SignupServiceImpl struct {
	eventRepository     repository.EventRepository
	signupRepository    repository.SignupRepository
	teamRepository      repository.TeamRepository
	activityService     ActivityService
	characterRepository repository.CharacterRepository
}

func NewSignupService() SignupService {
	return &SignupServiceImpl{
		signupRepository:    repository.NewSignupRepository(),
		eventRepository:     repository.NewEventRepository(),
		teamRepository:      repository.NewTeamRepository(),
		activityService:     NewActivityService(),
		characterRepository: repository.NewCharacterRepository(),
	}
}

func (r *SignupServiceImpl) SaveSignup(signup *repository.Signup) (*repository.Signup, error) {
	return r.signupRepository.SaveSignup(signup)
}

func (r *SignupServiceImpl) RemoveSignupForUser(userId int, eventId int) error {
	err := r.teamRepository.RemoveUserForEvent(userId, eventId)
	if err != nil {
		return err
	}
	return r.signupRepository.RemoveSignupForUser(userId, eventId)
}

func (r *SignupServiceImpl) GetSignupForUser(userId int, eventId int) (*repository.Signup, error) {
	return r.signupRepository.GetSignupForUser(userId, eventId)
}

type SignupWithUser struct {
	Signup   repository.Signup
	TeamUser *repository.TeamUser
}

func (r *SignupServiceImpl) GetSignupsForEvent(event *repository.Event) ([]*repository.Signup, error) {
	return r.signupRepository.GetSignupsForEvent(event.Id)
}

func (r *SignupServiceImpl) GetExtendedSignupsForEvent(event *repository.Event) (
	[]*repository.Signup, map[int]map[int]time.Duration, map[int]map[int]int, error) {
	signups, err := r.GetSignupsForEvent(event)
	if err != nil {
		return nil, nil, nil, err
	}
	userIds := make([]int, 0)
	for _, signup := range signups {
		userIds = append(userIds, signup.UserId)
	}
	userEventActivityCount, err := r.activityService.CalculateActiveTimesForUsers(userIds)
	if err != nil {
		return nil, nil, nil, err
	}
	highestCharacterLevels, err := r.characterRepository.GetHighestCharacterLevelForEventsForUsers(userIds)
	if err != nil {
		return nil, nil, nil, err
	}
	return signups, userEventActivityCount, highestCharacterLevels, nil
}
