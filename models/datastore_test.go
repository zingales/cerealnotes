package models_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/atmiguel/cerealnotes/models"
	"github.com/atmiguel/cerealnotes/test_util"
)

var postgresUrl = os.Getenv("DATABASE_URL_TEST")

const noteTable = "note"
const publicationTable = "publication"
const noteToPublicationTable = "note_to_publication_relationship"
const noteToCategoryTable = "note_to_category_relationship"
const userTable = "app_user"

var tables = []string{
	noteToPublicationTable,
	publicationTable,
	noteToCategoryTable,
	noteTable,
	userTable,
}

func ClearDatabase(db *models.DB) error {
	for _, val := range tables {
		if err := ClearTable(db, val); err != nil {
			return err
		}
	}
	return nil
}

func ClearTable(db *models.DB, table string) error {
	// db.Query() doesn't allow variables to replace columns or table names.
	sqlQuery := fmt.Sprintf(`TRUNCATE %s CASCADE;`, table)

	_, err := db.Exec(sqlQuery)
	if err != nil {
		return err
	}

	return nil
}

func TestUser(t *testing.T) {
	db, err := models.ConnectToDatabase(postgresUrl, 10)
	test_util.Ok(t, err)
	ClearTable(db, userTable)

	displayName := "boby"
	password := "aPassword"
	emailAddress := models.NewEmailAddress("thisIsMyOtherEmail@gmail.com")

	err = db.StoreNewUser(displayName, emailAddress, password)
	test_util.Ok(t, err)

	id, err := db.GetIdForUserWithEmailAddress(emailAddress)
	test_util.Ok(t, err)

	err = db.AuthenticateUserCredentials(emailAddress, password)
	test_util.Ok(t, err)

	userMap, err := db.GetAllUsersById()
	test_util.Ok(t, err)

	test_util.Equals(t, 1, len(userMap))

	user, isOk := userMap[id]
	test_util.Assert(t, isOk, "Expected UserId missing")

	test_util.Equals(t, displayName, user.DisplayName)
}

func TestNote(t *testing.T) {
	db, err := models.ConnectToDatabase(postgresUrl, 10)
	test_util.Ok(t, err)
	ClearTable(db, userTable)
	ClearTable(db, noteTable)

	displayName := "bob"
	password := "aPassword"
	emailAddress := models.NewEmailAddress("thisIsMyEmail@gmail.com")

	err = db.StoreNewUser(displayName, emailAddress, password)
	test_util.Ok(t, err)

	userId, err := db.GetIdForUserWithEmailAddress(emailAddress)
	test_util.Ok(t, err)

	note := &models.Note{AuthorId: userId, Content: "I'm a note", CreationTime: time.Now()}
	id, err := db.StoreNewNote(note)
	test_util.Ok(t, err)
	test_util.Assert(t, int64(id) > 0, "Note Id was not a valid index: "+strconv.Itoa(int(id)))

	notemap, err := db.GetMyUnpublishedNotes(userId)
	test_util.Ok(t, err)

	retrievedNote, isOk := notemap[id]
	test_util.Assert(t, isOk, "Expected NoteId missing")

	test_util.Equals(t, note.AuthorId, retrievedNote.AuthorId)
	test_util.Equals(t, note.Content, retrievedNote.Content)

	updatedContent := "some new coolenss"
	err = db.UpdateNoteContent(id, updatedContent)
	test_util.Ok(t, err)

	newNote, err := db.GetNoteById(id)
	test_util.Ok(t, err)
	test_util.Equals(t, updatedContent, newNote.Content)
	test_util.Equals(t, note.AuthorId, newNote.AuthorId)

	err = db.DeleteNoteById(id)
	test_util.Ok(t, err)
}

func TestPublication(t *testing.T) {
	db, err := models.ConnectToDatabase(postgresUrl, 10)
	test_util.Ok(t, err)
	ClearTable(db, userTable)
	ClearTable(db, noteTable)
	ClearTable(db, publicationTable)
	ClearTable(db, noteToPublicationTable)

	displayName := "bob"
	password := "aPassword"
	emailAddress := models.NewEmailAddress("thisIsMyEmail@gmail.com")

	err = db.StoreNewUser(displayName, emailAddress, password)
	test_util.Ok(t, err)

	userId, err := db.GetIdForUserWithEmailAddress(emailAddress)
	test_util.Ok(t, err)

	note := &models.Note{AuthorId: userId, Content: "I'm a note", CreationTime: time.Now()}
	id, err := db.StoreNewNote(note)
	test_util.Ok(t, err)
	test_util.Assert(t, int64(id) > 0, "Note Id was not a valid index: "+strconv.Itoa(int(id)))

	fmt.Println(userId)
	publicationToNotesById, err := db.GetAllPublishedNotesVisibleBy(userId)
	test_util.Ok(t, err)

	test_util.Equals(t, 0, len(publicationToNotesById))

	err = db.PublishNotes(userId)
	test_util.Ok(t, err)

	publicationToNotesById, err = db.GetAllPublishedNotesVisibleBy(userId)
	test_util.Equals(t, 1, len(publicationToNotesById))
}

func TestCategory(t *testing.T) {
	db, err := models.ConnectToDatabase(postgresUrl, 10)
	test_util.Ok(t, err)
	ClearTable(db, userTable)
	ClearTable(db, noteTable)
	ClearTable(db, noteToCategoryTable)

	displayName := "bob"
	password := "aPassword"
	emailAddress := models.NewEmailAddress("thisyetAnotherIsMyEmail@gmail.com")

	err = db.StoreNewUser(displayName, emailAddress, password)
	test_util.Ok(t, err)

	userId, err := db.GetIdForUserWithEmailAddress(emailAddress)
	test_util.Ok(t, err)

	note := &models.Note{AuthorId: userId, Content: "I'm a note", CreationTime: time.Now()}
	noteId, err := db.StoreNewNote(note)
	test_util.Ok(t, err)

	assignedCategory := models.META
	err = db.AssignNoteCategoryRelationship(noteId, assignedCategory)
	test_util.Ok(t, err)

	retrievedCategory, err := db.GetNoteCategory(noteId)
	test_util.Ok(t, err)
	test_util.Equals(t, assignedCategory, retrievedCategory)

	newAssignedCategory := models.PREDICTION
	err = db.AssignNoteCategoryRelationship(noteId, newAssignedCategory)
	test_util.Ok(t, err)

	newRetrievedCategory, err := db.GetNoteCategory(noteId)
	test_util.Ok(t, err)
	test_util.Equals(t, newAssignedCategory, newRetrievedCategory)

	err = db.DeleteNoteCategory(noteId)
	test_util.Ok(t, err)
}
