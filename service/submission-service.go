package service

import (
	"bpl/repository"
	"fmt"
)

type SubmissionService interface {
	GetSubmissions(eventId int) ([]*repository.Submission, error)
	SaveBulkSubmissions(submissions []*repository.Submission) ([]*repository.Submission, error)
	SaveSubmission(submission *repository.Submission, submitter *repository.User) (*repository.Submission, error)
	ReviewSubmission(submissionId int, submissionReview *repository.Submission, reviewer *repository.User) (*repository.Submission, error)
	GetSubmissionById(id int) (*repository.Submission, error)
	DeleteSubmission(submission *repository.Submission, user *repository.User) error
}

type SubmissionServiceImpl struct {
	submissionRepository repository.SubmissionRepository
	eventRepository      repository.EventRepository
}

func NewSubmissionService() SubmissionService {
	return &SubmissionServiceImpl{
		submissionRepository: repository.NewSubmissionRepository(),
		eventRepository:      repository.NewEventRepository(),
	}
}

func (e *SubmissionServiceImpl) GetSubmissions(eventId int) ([]*repository.Submission, error) {
	event, err := e.eventRepository.GetEventById(eventId, "Teams")
	if err != nil {
		return nil, err
	}
	return e.submissionRepository.GetSubmissionsForEvent(event)
}

func (e *SubmissionServiceImpl) SaveBulkSubmissions(submissions []*repository.Submission) ([]*repository.Submission, error) {
	persisted := make([]*repository.Submission, 0)
	for _, submission := range submissions {
		s, err := e.submissionRepository.SaveSubmission(submission)
		if err != nil {
			return nil, err
		}
		persisted = append(persisted, s)
	}
	return persisted, nil
}

func (e *SubmissionServiceImpl) SaveSubmission(submission *repository.Submission, submitter *repository.User) (*repository.Submission, error) {
	if submission.Id != 0 {
		existingSubmission, err := e.submissionRepository.GetSubmissionById(submission.Id)
		if err != nil {
			return nil, err
		}
		if existingSubmission.UserId != submitter.Id {
			return nil, fmt.Errorf("you are not allowed to edit this submission")
		}
		err = e.submissionRepository.RemoveMatchFromSubmission(existingSubmission)
		if err != nil {
			return nil, err
		}
		existingSubmission.ObjectiveId = submission.ObjectiveId
		existingSubmission.Timestamp = submission.Timestamp
		existingSubmission.Number = submission.Number
		existingSubmission.Proof = submission.Proof
		existingSubmission.Comment = submission.Comment
		existingSubmission.Extra = submission.Extra
		if existingSubmission.ApprovalStatus == repository.APPROVED {
			existingSubmission.ApprovalStatus = repository.PENDING
		}
		return e.submissionRepository.SaveSubmission(existingSubmission)
	}
	submission.ApprovalStatus = repository.PENDING
	submission.User = submitter
	submission.UserId = submitter.Id
	return e.submissionRepository.SaveSubmission(submission)
}

func (e *SubmissionServiceImpl) ReviewSubmission(submissionId int, submissionReview *repository.Submission, reviewer *repository.User) (*repository.Submission, error) {
	submission, err := e.submissionRepository.GetSubmissionById(submissionId)
	if err != nil {
		return nil, err
	}
	if submissionReview.ApprovalStatus == repository.APPROVED {
		err = e.submissionRepository.AddMatchToSubmission(submission)
	} else {
		err = e.submissionRepository.RemoveMatchFromSubmission(submission)
	}
	if err != nil {
		return nil, err
	}
	submission.ApprovalStatus = submissionReview.ApprovalStatus
	submission.ReviewComment = submissionReview.ReviewComment
	submission.ReviewerId = &reviewer.Id
	return e.submissionRepository.SaveSubmission(submission)
}

func (e *SubmissionServiceImpl) GetSubmissionById(id int) (*repository.Submission, error) {
	submission, err := e.submissionRepository.GetSubmissionById(id)
	if err != nil {
		return nil, err
	}
	return submission, nil
}

func (e *SubmissionServiceImpl) DeleteSubmission(submission *repository.Submission, user *repository.User) error {
	return e.submissionRepository.DeleteSubmission(submission.Id)
}
