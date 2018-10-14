package models

import (
	"errors"
	"time"
)

type NoteId int64

type Note struct {
	AuthorId     UserId    `json:"authorId"`
	Content      string    `json:"content"`
	CreationTime time.Time `json:"creationTime"`
}

var NoNoteFoundError = errors.New("No note with that information could be found")

//  DB methods

func (db *DB) StoreNewNote(
	note *Note,
) (NoteId, error) {

	authorId := int64(note.AuthorId)
	content := note.Content
	creationTime := note.CreationTime

	sqlQuery := `
		INSERT INTO note (author_id, content, creation_time)
		VALUES ($1, $2, $3)
		RETURNING id`

	var noteId int64 = 0
	if err := db.execOneResult(sqlQuery, &noteId, authorId, content, creationTime); err != nil {
		return 0, err
	}
	return NoteId(noteId), nil
}

func (db *DB) GetUsersNotes(userId UserId) (NotesById, error) {
	sqlQuery := `
		SELECT id, author_id, content, creation_time FROM note
		WHERE author_id = $1`

	noteMap, err := db.getNotesById(sqlQuery, int64(userId))
	if err != nil {
		return nil, err
	}

	return noteMap, nil
}

func (db *DB) GetAllPublishedNotesVisibleBy(userId UserId) (map[int64]NotesById, error) {

	sqlQueryIssueNumber := `
		SELECT COUNT(*) AS IssueNumber FROM publication
		WHERE publication.author_id = $1`

	var publictionIssueNumber int64
	if err := db.execOneResult(sqlQueryIssueNumber, &publictionIssueNumber, int64(userId)); err != nil {
		return nil, err
	}

	sqlQueryGetNotes := `
		SELECT
		note.id,
		note.author_id,
		note.content,
		note.creation_time,
		filtered_pubs.rank AS publication_issue
		FROM   (SELECT *,
					   Rank()
						 OVER(
						   partition BY pub.author_id
						   ORDER BY pub.creation_time)
				FROM   publication AS pub) filtered_pubs
			   INNER JOIN note_to_publication_relationship AS note2pub
					   ON note2pub.publication_id = filtered_pubs.id
			   INNER JOIN note
					   ON note.id = note2pub.note_id
		WHERE  rank <= ($1)`

	// sqlQueryGetNotes := `
	// 	SELECT
	// 	note.id,
	// 	note.author_id,
	// 	note.content,
	// 	note.creation_time,
	// 	note2cat.type      AS category,
	// 	filtered_pubs.rank AS publication_issue
	// 	FROM   (SELECT *,
	// 	               Rank()
	// 	                 OVER(
	// 	                   partition BY pub.author_id
	// 	                   ORDER BY pub.creation_time)
	// 	        FROM   publication AS pub) filtered_pubs
	// 	       INNER JOIN note_to_publication_relationship AS note2pub
	// 	               ON note2pub.publication_id = filtered_pubs.id
	// 	       INNER JOIN note
	// 	               ON note.id = note2pub.note_id
	// 	       LEFT OUTER JOIN note_to_category_relationship AS note2cat
	// 	                    ON note.id = note2cat.note_id
	// 	WHERE  rank <= ($1)`

	rows, err := db.Query(sqlQueryGetNotes, publictionIssueNumber)
	if err != nil {
		return nil, convertPostgresError(err)
	}

	defer rows.Close()

	pubToNotesById := make(map[int64]NotesById)

	for rows.Next() {
		var publicationNumber int64
		var noteId int64
		note := &Note{}
		if err := rows.Scan(&noteId, &note.AuthorId, &note.Content, &note.CreationTime, &publicationNumber); err != nil {
			return nil, err
		}

		noteMap, ok := pubToNotesById[publicationNumber]
		if !ok {
			pubToNotesById[publicationNumber] = make(map[NoteId]*Note)
		}

		noteMap[NoteId(noteId)] = note

	}
	return pubToNotesById, nil
}

func (db *DB) GetMyUnpublishedNotes(userId UserId) (NotesById, error) {
	sqlQuery := `
		SELECT id, author_id, content, creation_time FROM note
		LEFT OUTER JOIN note_to_publication_relationship AS note2pub
			ON note.id = note2pub.note_id
		WHERE note2pub.note_id is NULL AND note.author_id = $1`

	noteMap, err := db.getNotesById(sqlQuery, int64(userId))
	if err != nil {
		return nil, err
	}

	return noteMap, nil
}

func (db *DB) getNotesById(sqlQuery string, args ...interface{}) (NotesById, error) {

	noteMap := make(map[NoteId]*Note)

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, convertPostgresError(err)
	}
	defer rows.Close()

	for rows.Next() {
		var tempId int64
		tempNote := &Note{}
		if err := rows.Scan(&tempId, &tempNote.AuthorId, &tempNote.Content, &tempNote.CreationTime); err != nil {
			return nil, convertPostgresError(err)
		}

		noteMap[NoteId(tempId)] = tempNote
	}

	return noteMap, nil
}

func (db *DB) DeleteNoteById(noteId NoteId) error {
	sqlQuery := `
		DELETE FROM note
		WHERE id = $1`

	num, err := db.execNoResults(sqlQuery, int64(noteId))
	if err != nil {
		return err
	}

	if num == 0 {
		return NoNoteFoundError
	}

	if num != 1 {
		return errors.New("somehow more than 1 note was deleted")
	}

	return nil
}
