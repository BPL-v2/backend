package service

import (
	"bpl/client"
	"bpl/repository"
)

type AtlasService interface {
	SaveAtlasTrees(userId int, eventId int, treeHashes []client.LeagueAccountAtlasPassiveTree) error
	GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*repository.AtlasTree, err error)
	GetAtlasesForEventAndUser(userId int, eventId int) (atlas []*repository.AtlasTree, err error)
}

type AtlasServiceImpl struct {
	atlasRepository repository.AtlasRepository
	latestAtlas     map[int]map[int]map[int][32]byte
}

func NewAtlasService() AtlasService {
	return &AtlasServiceImpl{
		atlasRepository: repository.NewAtlasRepository(),
		latestAtlas:     make(map[int]map[int]map[int][32]byte),
	}
}

func (a *AtlasServiceImpl) initCache(userId int, eventId int) error {
	if a.latestAtlas[eventId] == nil {
		trees, err := a.atlasRepository.GetLatestTreesForEvent(eventId)
		if err != nil {
			return err
		}
		a.latestAtlas[eventId] = make(map[int]map[int][32]byte)
		for _, tree := range trees {
			if a.latestAtlas[eventId][tree.UserID] == nil {
				a.latestAtlas[eventId][tree.UserID] = map[int][32]byte{0: {}, 1: {}, 2: {}}
			}
			a.latestAtlas[eventId][tree.UserID][tree.Index] = tree.Nodes.GetHash()
		}
	}
	if a.latestAtlas[eventId][userId] == nil {
		a.latestAtlas[eventId][userId] = map[int][32]byte{0: {}, 1: {}, 2: {}}
	}
	return nil
}

func (a *AtlasServiceImpl) SaveAtlasTrees(userId int, eventId int, trees []client.LeagueAccountAtlasPassiveTree) error {
	if err := a.initCache(userId, eventId); err != nil {
		return err
	}
	hashValues := a.latestAtlas[eventId][userId]
	for index, tree := range trees {
		hashes := []int{}
		for _, hash := range tree.Hashes {
			hashes = append(hashes, hash)
		}
		if repository.PassiveNodes(hashes).GetHash() != hashValues[index] {
			err := a.atlasRepository.CreateAtlasTree(userId, eventId, index, hashes)
			if err != nil {
				return err
			}
			hashValues[index] = repository.PassiveNodes(hashes).GetHash()
		}
	}
	return nil
}

func (a *AtlasServiceImpl) GetLatestAtlasesForEventAndTeam(eventId int, teamId int) (atlas []*repository.AtlasTree, err error) {
	return a.atlasRepository.GetLatestAtlasesForEventAndTeam(eventId, teamId)
}

func (a *AtlasServiceImpl) GetAtlasesForEventAndUser(userId int, eventId int) (atlas []*repository.AtlasTree, err error) {
	return a.atlasRepository.GetAtlasesForEventAndUser(eventId, userId)
}
