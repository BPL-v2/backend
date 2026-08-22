package service

import "bpl/repository"

type TeamSheetService interface {
	GetEntriesForTeam(eventId int, teamId int) ([]*repository.TeamSheetEntry, error)
	GetEntryForUser(eventId int, userId int) (*repository.TeamSheetEntry, error)
	SaveEntry(entry *repository.TeamSheetEntry) (*repository.TeamSheetEntry, error)
}

type TeamSheetServiceImpl struct {
	teamSheetRepository repository.TeamSheetRepository
	teamRepository      repository.TeamRepository
}

func NewTeamSheetService() TeamSheetService {
	return &TeamSheetServiceImpl{
		teamSheetRepository: repository.NewTeamSheetRepository(),
		teamRepository:      repository.NewTeamRepository(),
	}
}

func (s *TeamSheetServiceImpl) GetEntriesForTeam(eventId int, teamId int) ([]*repository.TeamSheetEntry, error) {
	teamUsers, err := s.teamRepository.GetTeamUsersForTeam(teamId)
	if err != nil {
		return nil, err
	}
	userIds := make([]int, len(teamUsers))
	for i, teamUser := range teamUsers {
		userIds[i] = teamUser.UserId
	}
	return s.teamSheetRepository.GetEntriesForUsers(eventId, userIds)
}

func (s *TeamSheetServiceImpl) GetEntryForUser(eventId int, userId int) (*repository.TeamSheetEntry, error) {
	return s.teamSheetRepository.GetEntryForUser(eventId, userId)
}

func (s *TeamSheetServiceImpl) SaveEntry(entry *repository.TeamSheetEntry) (*repository.TeamSheetEntry, error) {
	return s.teamSheetRepository.SaveEntry(entry)
}
