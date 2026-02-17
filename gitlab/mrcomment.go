package gitlab

import (
	"github.com/getplumber/plumber/configuration"
	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ListMergeRequestNotes retrieves all notes (comments) on a merge request
func ListMergeRequestNotes(projectID int, mrIID int, token string, instanceURL string, conf *configuration.Configuration) ([]*gitlab.Note, error) {
	l := logger.WithFields(logrus.Fields{
		"action":    "ListMergeRequestNotes",
		"projectID": projectID,
		"mrIID":     mrIID,
	})

	glab, err := GetNewGitlabClient(token, instanceURL, conf)
	if err != nil {
		l.WithError(err).Error("Unable to get a Gitlab client")
		return nil, err
	}

	var allNotes []*gitlab.Note
	var perPage int64 = 100
	options := &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: perPage,
		},
	}

	for page := int64(1); ; page++ {
		options.Page = page
		notes, _, err := glab.Notes.ListMergeRequestNotes(projectID, int64(mrIID), options)
		if err != nil {
			l.WithError(err).Error("Failed to list merge request notes")
			return nil, err
		}

		allNotes = append(allNotes, notes...)

		if int64(len(notes)) < perPage {
			break
		}
	}

	l.WithField("noteCount", len(allNotes)).Debug("Fetched merge request notes")
	return allNotes, nil
}

// CreateMergeRequestNote creates a new note (comment) on a merge request
func CreateMergeRequestNote(projectID int, mrIID int, body string, token string, instanceURL string, conf *configuration.Configuration) (*gitlab.Note, error) {
	l := logger.WithFields(logrus.Fields{
		"action":    "CreateMergeRequestNote",
		"projectID": projectID,
		"mrIID":     mrIID,
	})

	glab, err := GetNewGitlabClient(token, instanceURL, conf)
	if err != nil {
		l.WithError(err).Error("Unable to get a Gitlab client")
		return nil, err
	}

	note, _, err := glab.Notes.CreateMergeRequestNote(projectID, int64(mrIID), &gitlab.CreateMergeRequestNoteOptions{
		Body: &body,
	})
	if err != nil {
		l.WithError(err).Error("Failed to create merge request note")
		return nil, err
	}

	l.WithField("noteID", note.ID).Debug("Created merge request note")
	return note, nil
}

// UpdateMergeRequestNote updates an existing note (comment) on a merge request
func UpdateMergeRequestNote(projectID int, mrIID int, noteID int, body string, token string, instanceURL string, conf *configuration.Configuration) (*gitlab.Note, error) {
	l := logger.WithFields(logrus.Fields{
		"action":    "UpdateMergeRequestNote",
		"projectID": projectID,
		"mrIID":     mrIID,
		"noteID":    noteID,
	})

	glab, err := GetNewGitlabClient(token, instanceURL, conf)
	if err != nil {
		l.WithError(err).Error("Unable to get a Gitlab client")
		return nil, err
	}

	note, _, err := glab.Notes.UpdateMergeRequestNote(projectID, int64(mrIID), int64(noteID), &gitlab.UpdateMergeRequestNoteOptions{
		Body: &body,
	})
	if err != nil {
		l.WithError(err).Error("Failed to update merge request note")
		return nil, err
	}

	l.WithField("noteID", note.ID).Debug("Updated merge request note")
	return note, nil
}
