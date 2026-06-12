package service

import (
	"bpl/repository"
	"errors"
)

type AchievementService interface {
	GetAllAchievements() ([]*repository.Achievement, error)
	GetAchievementById(id int) (*repository.Achievement, error)
	CreateAchievement(name, description string) (*repository.Achievement, error)
	UpdateAchievement(id int, name, description string) (*repository.Achievement, error)
	DeleteAchievement(id int) error

	UploadIcon(id int, icon []byte, mimeType string) error

	GetAllUserAchievements(userId *int) ([]*repository.UserAchievement, error)
	GrantAchievement(userId, achievementId int, grantedBy *int) error
	RevokeAchievement(userId, achievementId int) error

	SyncAchievements() error
}

type AchievementServiceImpl struct {
	achievementRepository repository.AchievementRepository
	characterRepository   repository.CharacterRepository
}

func NewAchievementService() AchievementService {
	return &AchievementServiceImpl{
		achievementRepository: repository.NewAchievementRepository(),
		characterRepository:   repository.NewCharacterRepository(),
	}
}

func (s *AchievementServiceImpl) GetAllAchievements() ([]*repository.Achievement, error) {
	return s.achievementRepository.GetAllAchievements()
}

func (s *AchievementServiceImpl) GetAchievementById(id int) (*repository.Achievement, error) {
	return s.achievementRepository.GetAchievementById(id)
}

func (s *AchievementServiceImpl) CreateAchievement(name, description string) (*repository.Achievement, error) {
	return s.achievementRepository.SaveAchievement(&repository.Achievement{
		Name:        name,
		Description: description,
		IsCustom:    true,
	})
}

func (s *AchievementServiceImpl) UpdateAchievement(id int, name, description string) (*repository.Achievement, error) {
	achievement, err := s.achievementRepository.GetAchievementById(id)
	if err != nil {
		return nil, err
	}
	if !achievement.IsCustom {
		return nil, errors.New("cannot modify system achievements")
	}
	achievement.Name = name
	achievement.Description = description
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
	nameToId := make(map[string]int, len(allAchievements))
	for _, a := range allAchievements {
		nameToId[a.Name] = a.Id
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
	for userId, chars := range characterMap {
		for _, name := range checkAchievements(chars) {
			id, ok := nameToId[name]
			if !ok {
				continue
			}
			grants = append(grants, &repository.UserAchievement{
				UserId:        userId,
				AchievementId: id,
			})
		}
	}
	return s.achievementRepository.SaveUserAchievements(grants)
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

func checkAchievements(chars []*repository.Character) []string {
	var results []string
	checks := []struct {
		name string
		pass bool
	}{
		{"Reached level 90", hasLevelNCharacter(90, chars)},
		{"Reached level 95", hasLevelNCharacter(95, chars)},
		{"Reached level 100", hasLevelNCharacter(100, chars)},
		{"Participated in an event", playedNLeagues(1, chars)},
		{"Played 5 leagues", playedNLeagues(5, chars)},
		{"Played 10 leagues", playedNLeagues(10, chars)},
		{"Played 5 different ascendancies", playedNDifferentAscendancies(5, chars)},
		{"Played 10 different ascendancies", playedNDifferentAscendancies(10, chars)},
	}
	for _, c := range checks {
		if c.pass {
			results = append(results, c.name)
		}
	}
	return results
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
