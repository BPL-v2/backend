package service

import (
	"bpl/repository"

	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	ApplicationStatusApplied    ApplicationStatus = "applied"
	ApplicationStatusAccepted   ApplicationStatus = "accepted"
	ApplicationStatusWaitlisted ApplicationStatus = "waitlisted"
	ApplicationStatusNone       ApplicationStatus = "none"
)

type EventStatus struct {
	TeamId                *int              `json:"team_id"`
	IsTeamLead            bool              `json:"is_team_lead" binding:"required"`
	ApplicationStatus     ApplicationStatus `json:"application_status" binding:"required"`
	NumberOfSignups       int               `json:"number_of_signups" binding:"required"`
	NumberOfSignupsBefore int               `json:"number_of_signups_before" binding:"required"`
}

type EventService interface {
	GetAllEvents(preloads ...string) ([]*repository.Event, error)
	CreateEvent(event *repository.Event) (*repository.Event, error)
	CreateEventWithoutCategory(event *repository.Event) (*repository.Event, error)
	GetEventById(eventId int, preloads ...string) (*repository.Event, error)
	GetCurrentEvent(preloads ...string) (*repository.Event, error)
	DeleteEvent(event *repository.Event) error
	GetEventByConditionId(conditionId int) (*repository.Event, error)
	GetEventStatus(event *repository.Event, user *repository.User) (*EventStatus, error)
}

type EventServiceImpl struct {
	eventRepository       repository.EventRepository
	scoringRuleRepository repository.ScoringRuleRepository
	objectiveRepository   repository.ObjectiveRepository
	teamService           TeamService
	signupService         SignupService
}

func NewEventService() EventService {
	return &EventServiceImpl{
		eventRepository:       repository.NewEventRepository(),
		scoringRuleRepository: repository.NewScoringRuleRepository(),
		objectiveRepository:   repository.NewObjectiveRepository(),
		teamService:           NewTeamService(),
		signupService:         NewSignupService(),
	}
}

func (e *EventServiceImpl) GetAllEvents(preloads ...string) ([]*repository.Event, error) {
	return e.eventRepository.FindAll(preloads...)
}

func (e *EventServiceImpl) CreateEvent(event *repository.Event) (*repository.Event, error) {
	if event.Id == 0 {
		event.Objectives = []*repository.Objective{{
			Name: "default",
		}}
	}
	if event.IsCurrent {
		err := e.eventRepository.InvalidateCurrentEvent()
		if err != nil {
			return nil, err
		}
	}
	return e.eventRepository.SaveEvent(event)
}
func (e *EventServiceImpl) CreateEventWithoutCategory(event *repository.Event) (*repository.Event, error) {
	if event.IsCurrent {
		err := e.eventRepository.InvalidateCurrentEvent()
		if err != nil {
			return nil, err
		}
	}
	return e.eventRepository.SaveEvent(event)
}

func (e *EventServiceImpl) GetEventById(eventId int, preloads ...string) (*repository.Event, error) {
	return e.eventRepository.GetEventById(eventId, preloads...)
}

func (e *EventServiceImpl) GetCurrentEvent(preloads ...string) (*repository.Event, error) {
	return e.eventRepository.GetCurrentEvent(preloads...)
}

func (e *EventServiceImpl) DeleteEvent(event *repository.Event) error {
	err := e.objectiveRepository.DeleteObjectivesByEventId(event.Id)
	if err != nil {
		return err
	}
	err = e.eventRepository.Delete(event)
	if err != nil {
		return err
	}
	return e.scoringRuleRepository.DeleteRulesForEvent(event.Id)
}

func (e *EventServiceImpl) GetEventByConditionId(conditionId int) (*repository.Event, error) {
	return e.eventRepository.GetEventByConditionId(conditionId)
}

func (e *EventServiceImpl) GetEventStatus(event *repository.Event, user *repository.User) (*EventStatus, error) {
	eventStatus := &EventStatus{
		ApplicationStatus: ApplicationStatusNone,
	}
	signups, err := e.signupService.GetSignupsForEvent(event)
	if err != nil {
		return nil, err
	}
	eventStatus.NumberOfSignups = len(signups)
	if user == nil {
		return eventStatus, nil
	}
	team, err := e.teamService.GetTeamForUser(event.Id, user.Id)
	if err != nil && err != gorm.ErrRecordNotFound {
		return eventStatus, err
	}
	count := 0
	for _, signup := range signups {
		count++
		if signup.UserId == user.Id {
			if count > event.MaxSize {
				eventStatus.ApplicationStatus = ApplicationStatusWaitlisted
			} else {
				eventStatus.ApplicationStatus = ApplicationStatusApplied
			}
			eventStatus.NumberOfSignupsBefore = count - 1
		}
	}

	if team != nil {
		eventStatus.TeamId = &team.TeamId
		eventStatus.IsTeamLead = team.IsTeamLead
		eventStatus.ApplicationStatus = ApplicationStatusAccepted
	}
	return eventStatus, nil
}
