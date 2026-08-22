package service

import "bpl/repository"

type TeamSheetService interface {
	GetEntriesForTeam(eventId int, teamId int) ([]*repository.TeamSheetEntry, error)
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
	teamUserIds := make(map[int]bool, len(teamUsers))
	for _, teamUser := range teamUsers {
		teamUserIds[teamUser.UserId] = true
	}
	entries, err := s.teamSheetRepository.GetEntriesForEvent(eventId)
	if err != nil {
		return nil, err
	}
	teamEntries := make([]*repository.TeamSheetEntry, 0)
	for _, entry := range entries {
		if teamUserIds[entry.UserId] {
			teamEntries = append(teamEntries, entry)
		}
	}
	return teamEntries, nil
}

func (s *TeamSheetServiceImpl) SaveEntry(entry *repository.TeamSheetEntry) (*repository.TeamSheetEntry, error) {
	return s.teamSheetRepository.SaveEntry(entry)
}
