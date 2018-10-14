package models

import (
	"errors"
)

type NoteCategory int

const (
	MARGINALIA NoteCategory = iota
	META
	QUESTIONS
	PREDICTIONS
)

var categoryStrings = [...]string{
	"marginalia",
	"meta",
	"questions",
	"predictions",
}

var CannotDeserializeNoteCategoryStringError = errors.New("String does not correspond to a Note Category")
var NoteAlreadyContainsCategoryError = errors.New("NoteId already has a category stored for it")

func DeserializeNoteCategory(input string) (NoteCategory, error) {
	for i := 0; i < len(categoryStrings); i++ {
		if input == categoryStrings[i] {
			return NoteCategory(i), nil
		}
	}
	return 0, CannotDeserializeNoteCategoryStringError
}

func (category NoteCategory) String() string {

	if category < MARGINALIA || category > PREDICTIONS {
		return "Unknown"
	}

	return categoryStrings[category]
}

func (db *DB) GetNoteCategory(noteId NoteId) (NoteCategory, error) {
	sqlQuery := `
		SELECT category FROM note_to_category_relationship
		WHERE note_id = $1`
	var categoryString string
	if err := db.execOneResult(sqlQuery, &categoryString, int64(noteId)); err != nil {
		return 0, err
	}
	category, err := DeserializeNoteCategory(categoryString)
	if err != nil {
		return 0, err
	}
	return category, nil
}

func (db *DB) AssignNoteCategoryRelationship(noteId NoteId, category NoteCategory) error {
	sqlQuery := `
		INSERT INTO note_to_category_relationship (note_id, category)
		VALUES ($1, $2)
		ON CONFLICT (note_id) DO UPDATE SET category = ($2)`
	rowsAffected, err := db.execNoResults(sqlQuery, int64(noteId), category.String())
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return NoNoteFoundError
	}
	if rowsAffected > 1 {
		return TooManyRowsAffectedError
	}
	return nil
}

func (db *DB) DeleteNoteCategory(noteId NoteId) error {
	sqlQuery := `
		DELETE FROM note_to_category_relationship
		WHERE note_id = $1`
	num, err := db.execNoResults(sqlQuery, int64(noteId))
	if err != nil {
		return err
	}
	if num == 0 {
		return NoNoteFoundError
	}
	if num != 1 {
		return TooManyRowsAffectedError
	}
	return nil
}
