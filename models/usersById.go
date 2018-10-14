package models

import (
	"encoding/json"
	"fmt"
)

type UsersById map[UserId]*User

func (notesById UsersById) ToJson() ([]byte, error) {
	// json doesn't support int indexed maps
	notesByIdString := make(map[string]User, len(notesById))

	for id, note := range notesById {
		notesByIdString[fmt.Sprint(id)] = *note
	}

	return json.Marshal(notesByIdString)
}
