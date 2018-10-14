package models

import (
	"database/sql"
)

// ConnectToDatabase also pings the database to ensure a working connection.
func ConnectToDatabase(databaseUrl string) (*DB, error) {
	tempDb, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		return nil, err
	}

	if err := tempDb.Ping(); err != nil {
		return nil, err
	}

	return &DB{tempDb}, nil
}

type Datastore interface {
	StoreNewNote(*Note) (NoteId, error)
	StoreNewNoteCategoryRelationship(NoteId, NoteCategory) error
	StoreNewUser(string, *EmailAddress, string) error
	AuthenticateUserCredentials(*EmailAddress, string) error
	GetIdForUserWithEmailAddress(*EmailAddress) (UserId, error)
	GetUsersNotes(UserId) (NotesById, error)
	DeleteNoteById(NoteId) error
	GetMyUnpublishedNotes(UserId) (NotesById, error)
	GetAllUsersById() (UsersById, error)
	GetAllPublishedNotesVisibleBy(UserId) (map[int64]NotesById, error)
  PublishNotes(UserId) error
	StoreNewPublication(*Publication) (PublicationId, error)
}

type DB struct {
	*sql.DB
}
