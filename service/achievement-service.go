package service

import (
	"bpl/repository"
	"bpl/utils"
	"context"
	"errors"
	"fmt"
	"time"
)

type AchievementService interface {
	GetAllAchievements() ([]*repository.Achievement, error)
	GetAchievementById(id int) (*repository.Achievement, error)
	CreateAchievement(name, description string, eventId *int) (*repository.Achievement, error)
	UpdateAchievement(id int, name, description string, eventId *int) (*repository.Achievement, error)
	DeleteAchievement(id int) error

	UploadIcon(id int, icon []byte, mimeType string) error

	GetAllUserAchievements(userId *int) ([]*repository.UserAchievement, error)
	GrantAchievement(userId, achievementId int, grantedBy *int) error
	RevokeAchievement(userId, achievementId int) error

	SyncAchievements() error
	SyncAchievementsLoop(ctx context.Context, sleepDuration time.Duration)
}

type AchievementServiceImpl struct {
	achievementRepository repository.AchievementRepository
	characterRepository   repository.CharacterRepository
	teamRepository        repository.TeamRepository
	submissionRepository  repository.SubmissionRepository
}

func NewAchievementService() AchievementService {
	return &AchievementServiceImpl{
		achievementRepository: repository.NewAchievementRepository(),
		characterRepository:   repository.NewCharacterRepository(),
		teamRepository:        repository.NewTeamRepository(),
		submissionRepository:  repository.NewSubmissionRepository(),
	}
}

func (s *AchievementServiceImpl) GetAllAchievements() ([]*repository.Achievement, error) {
	return s.achievementRepository.GetAllAchievements()
}

func (s *AchievementServiceImpl) GetAchievementById(id int) (*repository.Achievement, error) {
	return s.achievementRepository.GetAchievementById(id)
}

func (s *AchievementServiceImpl) CreateAchievement(name, description string, eventId *int) (*repository.Achievement, error) {
	return s.achievementRepository.SaveAchievement(&repository.Achievement{
		Name:        name,
		Description: description,
		IsCustom:    true,
		EventId:     eventId,
	})
}

func (s *AchievementServiceImpl) UpdateAchievement(id int, name, description string, eventId *int) (*repository.Achievement, error) {
	achievement, err := s.achievementRepository.GetAchievementById(id)
	if err != nil {
		return nil, err
	}
	if !achievement.IsCustom {
		return nil, errors.New("cannot modify system achievements")
	}
	achievement.Name = name
	achievement.Description = description
	achievement.EventId = eventId
	return s.achievementRepository.SaveAchievement(achievement)
}

func (s *AchievementServiceImpl) DeleteAchievement(id int) error {
	achievement, err := s.achievementRepository.GetAchievementById(id)
	if err != nil {
		return err
	}
	if !achievement.IsCustom {
		return errors.New("cannot delete system achievements")
	}
	return s.achievementRepository.DeleteAchievement(id)
}

func (s *AchievementServiceImpl) UploadIcon(id int, icon []byte, mimeType string) error {
	return s.achievementRepository.SaveIcon(id, icon, mimeType)
}

func (s *AchievementServiceImpl) GetAllUserAchievements(userId *int) ([]*repository.UserAchievement, error) {
	return s.achievementRepository.GetAllUserAchievements(userId)
}

func (s *AchievementServiceImpl) GrantAchievement(userId, achievementId int, grantedBy *int) error {
	return s.achievementRepository.SaveUserAchievement(&repository.UserAchievement{
		UserId:        userId,
		AchievementId: achievementId,
		GrantedBy:     grantedBy,
	})
}

func (s *AchievementServiceImpl) RevokeAchievement(userId, achievementId int) error {
	return s.achievementRepository.DeleteUserAchievement(userId, achievementId)
}

func (s *AchievementServiceImpl) SyncAchievements() error {
	allAchievements, err := s.achievementRepository.GetAllAchievements()
	if err != nil {
		return err
	}

	characters, err := s.characterRepository.GetAllHighestLevelCharactersForEachEventAndUser()
	if err != nil {
		return err
	}
	characterMap := make(map[int][]*repository.Character)
	for _, char := range characters {
		if char.UserId != nil {
			characterMap[*char.UserId] = append(characterMap[*char.UserId], char)
		}
	}

	var grants []*repository.UserAchievement
	for _, achievement := range allAchievements {
		if achievement.AutoCheckKey == nil {
			continue
		}
		check, ok := achievementChecks[*achievement.AutoCheckKey]
		if !ok {
			fmt.Printf("Unknown achievement auto_check_key %q for achievement %d\n", *achievement.AutoCheckKey, achievement.Id)
			continue
		}
		userIds, err := check(s, achievement.EventId, characterMap)
		if err != nil {
			return err
		}
		for _, userId := range userIds {
			grants = append(grants, &repository.UserAchievement{
				UserId:        userId,
				AchievementId: achievement.Id,
			})
		}
	}
	return s.achievementRepository.SaveUserAchievements(grants)
}

func (s *AchievementServiceImpl) SyncAchievementsLoop(ctx context.Context, sleepDuration time.Duration) {
	ticker := time.NewTicker(sleepDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.SyncAchievements(); err != nil {
				fmt.Printf("Failed to sync achievements: %v\n", err)
			}
		}
	}
}

var baseClasses = map[string]bool{
	"Scion":    true,
	"Marauder": true,
	"Ranger":   true,
	"Witch":    true,
	"Shadow":   true,
	"Duelist":  true,
	"Templar":  true,
}

type achievementCheckFunc func(s *AchievementServiceImpl, eventId *int, characterMap map[int][]*repository.Character) ([]int, error)

func characterStatCheck(fn func([]*repository.Character) bool) achievementCheckFunc {
	return func(_ *AchievementServiceImpl, eventId *int, characterMap map[int][]*repository.Character) ([]int, error) {
		var userIds []int
		for userId, chars := range characterMap {
			if eventId != nil {
				chars = utils.Filter(chars, func(c *repository.Character) bool { return c.EventId == *eventId })
			}
			if fn(chars) {
				userIds = append(userIds, userId)
			}
		}
		return userIds, nil
	}
}

var achievementChecks = map[repository.AchievementCheckKey]achievementCheckFunc{
	repository.CheckLevel90:              characterStatCheck(func(c []*repository.Character) bool { return hasLevelNCharacter(90, c) }),
	repository.CheckLevel95:              characterStatCheck(func(c []*repository.Character) bool { return hasLevelNCharacter(95, c) }),
	repository.CheckLevel100:             characterStatCheck(func(c []*repository.Character) bool { return hasLevelNCharacter(100, c) }),
	repository.CheckParticipatedInEvent:  characterStatCheck(func(c []*repository.Character) bool { return playedNLeagues(1, c) }),
	repository.CheckPlayed5Leagues:       characterStatCheck(func(c []*repository.Character) bool { return playedNLeagues(5, c) }),
	repository.CheckPlayed10Leagues:      characterStatCheck(func(c []*repository.Character) bool { return playedNLeagues(10, c) }),
	repository.CheckPlayed5Ascendancies:  characterStatCheck(func(c []*repository.Character) bool { return playedNDifferentAscendancies(5, c) }),
	repository.CheckPlayed10Ascendancies: characterStatCheck(func(c []*repository.Character) bool { return playedNDifferentAscendancies(10, c) }),
	repository.CheckTeamlead: func(s *AchievementServiceImpl, eventId *int, _ map[int][]*repository.Character) ([]int, error) {
		if eventId == nil {
			return s.teamRepository.GetAllTeamLeadUserIds()
		}
		teamUsers, err := s.teamRepository.GetTeamLeadsForEvent(*eventId)
		if err != nil {
			return nil, err
		}
		return utils.Map(teamUsers, func(tu *repository.TeamUser) int { return tu.UserId }), nil
	},
	repository.CheckSubmittedBounty: func(s *AchievementServiceImpl, eventId *int, _ map[int][]*repository.Character) ([]int, error) {
		return s.submissionRepository.GetApprovedSubmissionUserIds(eventId)
	},
}

func hasLevelNCharacter(level int, chars []*repository.Character) bool {
	for _, char := range chars {
		if char.Level >= level {
			return true
		}
	}
	return false
}

func playedNLeagues(n int, chars []*repository.Character) bool {
	return len(chars) >= n
}

func playedNDifferentAscendancies(n int, chars []*repository.Character) bool {
	ascendancySet := make(map[string]bool)
	for _, char := range chars {
		if !baseClasses[char.Ascendancy] {
			ascendancySet[char.Ascendancy] = true
		}
	}
	return len(ascendancySet) >= n
}
