package main_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/atmiguel/cerealnotes/handlers"
	"github.com/atmiguel/cerealnotes/models"
	"github.com/atmiguel/cerealnotes/paths"
	"github.com/atmiguel/cerealnotes/routers"
	"github.com/atmiguel/cerealnotes/test_util"
)

func TestLoginOrSignUpPage(t *testing.T) {
	mockDb := &DiyMockDataStore{}
	env := &handlers.Environment{mockDb, []byte("")}

	server := httptest.NewServer(routers.DefineRoutes(env))
	defer server.Close()

	resp, err := http.Get(server.URL)
	test_util.Ok(t, err)
	test_util.Equals(t, http.StatusOK, resp.StatusCode)
}

func TestAuthenticatedFlow(t *testing.T) {
	mockDb := &DiyMockDataStore{}
	env := &handlers.Environment{mockDb, []byte("")}

	server := httptest.NewServer(routers.DefineRoutes(env))
	defer server.Close()

	// Create testing client
	client := &http.Client{}
	{
		jar, err := cookiejar.New(&cookiejar.Options{})

		if err != nil {
			panic(err)
		}

		client.Jar = jar
	}

	// Test login
	userIdAsInt := int64(1)
	{
		theEmail := "justsomeemail@gmail.com"
		thePassword := "worldsBestPassword"

		mockDb.Func_AuthenticateUserCredentials = func(email *models.EmailAddress, password string) error {
			if email.String() == theEmail && password == thePassword {
				return nil
			}

			return models.CredentialsNotAuthorizedError
		}

		mockDb.Func_GetIdForUserWithEmailAddress = func(email *models.EmailAddress) (models.UserId, error) {
			return models.UserId(userIdAsInt), nil
		}

		userValues := map[string]string{"emailAddress": theEmail, "password": thePassword}

		userJsonValue, _ := json.Marshal(userValues)

		resp, err := client.Post(server.URL+paths.SessionApi, "application/json", bytes.NewBuffer(userJsonValue))

		test_util.Ok(t, err)

		test_util.Equals(t, http.StatusCreated, resp.StatusCode)
	}

	// Test Add Note
	noteIdAsInt := int64(33)
	content := "Duuude I just said something cool"
	{
		noteValues := map[string]string{"content": content}

		mockDb.Func_StoreNewNote = func(*models.Note) (models.NoteId, error) {
			return models.NoteId(noteIdAsInt), nil
		}

		noteJsonValue, _ := json.Marshal(noteValues)

		resp, err := client.Post(server.URL+paths.NoteApi, "application/json", bytes.NewBuffer(noteJsonValue))
		test_util.Ok(t, err)
		test_util.Equals(t, http.StatusCreated, resp.StatusCode)

		type NoteResponse struct {
			NoteId int64 `json:"noteId"`
		}

		jsonNoteReponse := &NoteResponse{}

		err = json.NewDecoder(resp.Body).Decode(jsonNoteReponse)
		test_util.Ok(t, err)

		test_util.Equals(t, noteIdAsInt, jsonNoteReponse.NoteId)
		resp.Body.Close()
	}

	// Test get notes
	{
		mockDb.Func_GetMyUnpublishedNotes = func(userId models.UserId) (models.NotesById, error) {
			if userIdAsInt != int64(userId) {
				return nil, errors.New("Invalid userId passed in")
			}

			return models.NotesById(map[models.NoteId]*models.Note{
				models.NoteId(noteIdAsInt): &models.Note{
					AuthorId:     models.UserId(userIdAsInt),
					Content:      content,
					CreationTime: time.Now(),
				},
			}), nil

		}

		mockDb.Func_GetAllPublishedNotesVisibleBy = func(userId models.UserId) (map[int64]models.NotesById, error) {
			if userIdAsInt != int64(userId) {
				return nil, errors.New("Invalid userId passed in")
			}

			return map[int64]models.NotesById{
				1: models.NotesById(map[models.NoteId]*models.Note{
					models.NoteId(44): &models.Note{
						AuthorId:     models.UserId(99),
						Content:      "another note",
						CreationTime: time.Now(),
					},
				}),
			}, nil

		}

		resp, err := client.Get(server.URL + paths.NoteApi)
		test_util.Ok(t, err)
		test_util.Equals(t, http.StatusOK, resp.StatusCode)

		// TODO when we implement a real get notes feature we should enhance this code.
	}

	// Test Add category
	{
		type NoteCategoryForm struct {
			NoteCategory string `json:"category"`
		}

		metaNoteCategory := models.META

		categoryForm := &NoteCategoryForm{NoteCategory: metaNoteCategory.String()}

		mockDb.Func_StoreNewNoteCategoryRelationship = func(noteId models.NoteId, cat models.NoteCategory) error {
			if int64(noteId) == noteIdAsInt && cat == metaNoteCategory {
				return nil
			}

			return errors.New("Incorrect Data Arrived")
		}

		jsonValue, _ := json.Marshal(categoryForm)

    resp, err := sendPutRequest(client, server.URL+paths.NoteCategoryApi+"?id="+strconv.FormatInt(noteIdAsInt, 10), "application/json", bytes.NewBuffer(jsonValue))
		test_util.Ok(t, err)
		test_util.Equals(t, http.StatusCreated, resp.StatusCode)


	// Test publish notes
	{
		mockDb.Func_PublishNotes = func(userId models.UserId) error {
			return nil
		}
		// publish new api
		resp, err := client.Post(server.URL+paths.PublicationApi, "", nil)
		printBody(resp)
		ok(t, err)
		equals(t, http.StatusCreated, resp.StatusCode)
	}

	// Delete note
	{
		mockDb.Func_GetUsersNotes = func(userId models.UserId) (models.NotesById, error) {
			return models.NotesById(map[models.NoteId]*models.Note{
				models.NoteId(noteIdAsInt): &models.Note{
					AuthorId:     models.UserId(userIdAsInt),
					Content:      content,
					CreationTime: time.Now(),
				},
			}), nil
		}

		mockDb.Func_DeleteNoteById = func(noteid models.NoteId) error {
			if int64(noteid) == noteIdAsInt {
				return nil
			}

			return errors.New("Somehow you didn't get the correct error")
		}

		resp, err := sendDeleteRequest(client, server.URL+paths.NoteApi+"?id="+strconv.FormatInt(noteIdAsInt, 10))
		test_util.Ok(t, err)
		// printBody(resp)

		test_util.Equals(t, http.StatusOK, resp.StatusCode)
	}

}

// func sendDeleteRequest(client *http.Client, myUrl string, contentType string, body io.Reader) (resp *http.Response, err error) {
func sendDeleteRequest(client *http.Client, myUrl string) (resp *http.Response, err error) {

	req, err := http.NewRequest("DELETE", myUrl, nil)

	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

func sendPutRequest(client *http.Client, myUrl string, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("PUT", myUrl, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	return client.Do(req)
}

func printBody(resp *http.Response) {
	buf, bodyErr := ioutil.ReadAll(resp.Body)
	if bodyErr != nil {
		fmt.Print("bodyErr ", bodyErr.Error())
		return
	}

	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
	fmt.Printf("BODY: %q", rdr1)
	resp.Body = rdr2
}

// Helpers

type DiyMockDataStore struct {
	Func_StoreNewNote                     func(*models.Note) (models.NoteId, error)
	Func_StoreNewNoteCategoryRelationship func(models.NoteId, models.NoteCategory) error
	Func_StoreNewUser                     func(string, *models.EmailAddress, string) error
	Func_AuthenticateUserCredentials      func(*models.EmailAddress, string) error
	Func_GetIdForUserWithEmailAddress     func(*models.EmailAddress) (models.UserId, error)
	Func_GetUsersNotes                    func(models.UserId) (models.NotesById, error)
	Func_DeleteNoteById                   func(models.NoteId) error
	Func_GetMyUnpublishedNotes            func(models.UserId) (models.NotesById, error)
	Func_GetAllUsersById                  func() (models.UsersById, error)
	Func_GetAllPublishedNotesVisibleBy    func(models.UserId) (map[int64]models.NotesById, error)
  Func_PublishNotes                     func(models.UserId) error
	Func_StoreNewPublication              func(*models.Publication) (models.PublicationId, error)
}

func (mock *DiyMockDataStore) StoreNewNote(note *models.Note) (models.NoteId, error) {
	return mock.Func_StoreNewNote(note)
}


func (mock *DiyMockDataStore) StoreNewNoteCategoryRelationship(noteId models.NoteId, cat models.NoteCategory) error {
	return mock.Func_StoreNewNoteCategoryRelationship(noteId, cat)
}

func (mock *DiyMockDataStore) StoreNewUser(str1 string, email *models.EmailAddress, str2 string) error {
	return mock.Func_StoreNewUser(str1, email, str2)
}

func (mock *DiyMockDataStore) AuthenticateUserCredentials(email *models.EmailAddress, str string) error {
	return mock.Func_AuthenticateUserCredentials(email, str)
}

func (mock *DiyMockDataStore) GetIdForUserWithEmailAddress(email *models.EmailAddress) (models.UserId, error) {
	return mock.Func_GetIdForUserWithEmailAddress(email)
}

func (mock *DiyMockDataStore) GetUsersNotes(userId models.UserId) (models.NotesById, error) {
	return mock.Func_GetUsersNotes(userId)
}

func (mock *DiyMockDataStore) DeleteNoteById(noteId models.NoteId) error {
	return mock.Func_DeleteNoteById(noteId)
}
  
func (mock *DiyMockDataStore) GetMyUnpublishedNotes(userId models.UserId) (models.NotesById, error) {
	return mock.Func_GetMyUnpublishedNotes(userId)
}

func (mock *DiyMockDataStore) GetAllUsersById() (models.UsersById, error) {
	return mock.Func_GetAllUsersById()
}

func (mock *DiyMockDataStore) GetAllPublishedNotesVisibleBy(userId models.UserId) (map[int64]models.NotesById, error) {
	return mock.Func_GetAllPublishedNotesVisibleBy(userId)
}

func (mock *DiyMockDataStore) PublishNotes(userId models.UserId) error {
	return mock.Func_PublishNotes(userId)
}

func (mock *DiyMockDataStore) StoreNewPublication(publication *models.Publication) (models.PublicationId, error) {
	return mock.Func_StoreNewPublication(publication)
}