package models

import (
	"errors"
	"strings"
	"time"
)

type NoteId int64

type Category int

const (
	MARGINALIA Category = iota
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

var UnDeserializeableCategoryStringError = errors.New("String does not correspond to a Note Category")

func DeserializeCategory(input string) (Category, error) {
	for i := 0; i < len(categoryStrings); i++ {
		if input == categoryStrings[i] {
			return Category(i), nil
		}
	}
	return MARGINALIA, UnDeserializeableCategoryStringError
}

func (category Category) String() string {
	return categoryStrings[category]
}

type Note struct {
	Id           int64     `json:"id"`
	AuthorId     UserId    `json:"authorId"`
	Content      string    `json:"content"`
	CreationTime time.Time `json:"creationTime"`
}

func CreateNewNote(userId UserId, content string) *Note {
	return &Note{
		Id:           -1,
		AuthorId:     userId,
		Content:      content,
		CreationTime: time.Now().UTC(),
	}
}
