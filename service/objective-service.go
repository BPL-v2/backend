package service

import (
	"bpl/parser"
	"bpl/repository"
	"bpl/utils"
)

type ObjectiveService interface {
	CreateObjective(objective *repository.Objective, presetIds []int) (*repository.Objective, error)
	DeleteObjective(objectiveId int) error
	GetObjectiveById(objectiveId int, preloads ...string) (*repository.Objective, error)
	GetParser(eventId int, ignoreTime bool) (*parser.ItemChecker, error)
	StartSync(objectiveIds []int) error
	SetSynced(objectiveIds []int) error
	GetObjectiveTreeForEvent(eventId int, preloads ...string) (*repository.Objective, error)
	GetObjectivesForEvent(eventId int, preloads ...string) ([]*repository.Objective, error)
	GetAllObjectives(preloads ...string) ([]*repository.Objective, error)
	DuplicateObjectives(oldEventId int, newEventId int, ruleMap map[int]*repository.ScoringRule) error
	CopyObjectiveSubtree(sourceObjectiveId int, targetParentId int) (*repository.Objective, error)
}

type ObjectiveServiceImpl struct {
	objectiveRepository   repository.ObjectiveRepository
	scoringRuleRepository repository.ScoringRuleRepository
}

func NewObjectiveService() ObjectiveService {
	return &ObjectiveServiceImpl{
		objectiveRepository:   repository.NewObjectiveRepository(),
		scoringRuleRepository: repository.NewScoringRuleRepository(),
	}
}

func (e *ObjectiveServiceImpl) CreateObjective(objective *repository.Objective, ruleIds []int) (*repository.Objective, error) {
	var err error
	objective, err = e.objectiveRepository.SaveObjective(objective)
	if err != nil {
		return nil, err
	}
	err = e.objectiveRepository.AssociateScoringRules(objective.Id, ruleIds)
	if err != nil {
		return nil, err
	}
	return objective, nil
}

func (e *ObjectiveServiceImpl) DeleteObjective(objectiveId int) error {
	return e.objectiveRepository.DeleteObjective(objectiveId)
}

func (e *ObjectiveServiceImpl) GetObjectiveById(objectiveId int, preloads ...string) (*repository.Objective, error) {
	return e.objectiveRepository.GetObjectiveById(objectiveId, preloads...)
}

func (e *ObjectiveServiceImpl) GetParser(eventId int, ignoreTime bool) (*parser.ItemChecker, error) {
	objectives, err := e.GetObjectivesForEvent(eventId, "ScoringRules")
	if err != nil {
		return nil, err
	}
	return parser.NewItemChecker(objectives, ignoreTime)
}

func (e *ObjectiveServiceImpl) StartSync(objectiveIds []int) error {
	return e.objectiveRepository.StartSync(objectiveIds)
}

func (e *ObjectiveServiceImpl) SetSynced(objectiveIds []int) error {
	return e.objectiveRepository.FinishSync(objectiveIds)
}

func (e *ObjectiveServiceImpl) GetObjectiveTreeForEvent(eventId int, preloads ...string) (*repository.Objective, error) {
	return e.objectiveRepository.GetObjectivesByEventId(eventId, preloads...)
}

func (e *ObjectiveServiceImpl) GetObjectivesForEvent(eventId int, preloads ...string) ([]*repository.Objective, error) {
	return e.objectiveRepository.GetObjectivesByEventIdFlat(eventId, preloads...)
}

func (e *ObjectiveServiceImpl) GetAllObjectives(preloads ...string) ([]*repository.Objective, error) {
	return e.objectiveRepository.GetAllObjectives(preloads...)
}

func (e *ObjectiveServiceImpl) CopyObjectiveSubtree(sourceObjectiveId int, targetParentId int) (*repository.Objective, error) {
	source, err := e.objectiveRepository.GetObjectiveById(sourceObjectiveId)
	if err != nil {
		return nil, err
	}

	allObjectives, err := e.objectiveRepository.GetObjectivesByEventIdFlat(source.EventId)
	if err != nil {
		return nil, err
	}

	childrenMap := make(map[int][]int)
	for _, obj := range allObjectives {
		if obj.ParentId != nil {
			childrenMap[*obj.ParentId] = append(childrenMap[*obj.ParentId], obj.Id)
		}
	}

	subtreeIds := make(map[int]struct{})
	queue := []int{sourceObjectiveId}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		subtreeIds[id] = struct{}{}
		queue = append(queue, childrenMap[id]...)
	}

	targetParent, err := e.objectiveRepository.GetObjectiveById(targetParentId)
	if err != nil {
		return nil, err
	}

	oldToNew := make(map[int]*repository.Objective)
	for _, obj := range allObjectives {
		if _, ok := subtreeIds[obj.Id]; !ok {
			continue
		}
		newObj := *obj
		newObj.Id = 0
		newObj.EventId = targetParent.EventId
		newObj.ScoringRules = nil
		oldToNew[obj.Id] = &newObj
	}

	newObjectives := utils.Values(oldToNew)
	_, err = e.objectiveRepository.SaveObjectives(newObjectives)
	if err != nil {
		return nil, err
	}

	for oldId, newObj := range oldToNew {
		if oldId == sourceObjectiveId {
			newObj.ParentId = &targetParentId
		} else if newObj.ParentId != nil {
			if newParent, ok := oldToNew[*newObj.ParentId]; ok {
				newObj.ParentId = &newParent.Id
			}
		}
	}

	_, err = e.objectiveRepository.SaveObjectives(newObjectives)
	if err != nil {
		return nil, err
	}

	return oldToNew[sourceObjectiveId], nil
}

func (e *ObjectiveServiceImpl) DuplicateObjectives(oldEventId int, newEventId int, ruleMap map[int]*repository.ScoringRule) error {
	objectives, err := e.objectiveRepository.GetObjectivesByEventIdFlat(oldEventId, "ScoringRules")
	if err != nil {
		return err
	}
	newObjectiveMap := make(map[int]*repository.Objective)
	for _, objective := range objectives {
		newObjective := *objective
		oldId := newObjective.Id
		newObjective.Id = 0
		newObjective.EventId = newEventId

		newRules := utils.Filter(utils.Map(objective.ScoringRules, func(rule *repository.ScoringRule) *repository.ScoringRule {
			if newRule, ok := ruleMap[rule.Id]; ok {
				return newRule
			}
			return nil
		}), func(rule *repository.ScoringRule) bool { return rule != nil })
		newObjective.ScoringRules = newRules

		newObjectiveMap[oldId] = &newObjective
	}
	_, err = e.objectiveRepository.SaveObjectives(utils.Values(newObjectiveMap))
	if err != nil {
		return err
	}
	for _, objective := range objectives {
		newObjective := newObjectiveMap[objective.Id]
		if newObjective == nil || objective.ParentId == nil || newObjectiveMap[*objective.ParentId] == nil {
			continue
		}
		newObjective.ParentId = &newObjectiveMap[*objective.ParentId].Id
	}
	_, err = e.objectiveRepository.SaveObjectives(utils.Values(newObjectiveMap))
	return err

}
