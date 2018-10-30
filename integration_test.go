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
	mockDb := &MockDataStore{}
	env := &handlers.Environment{mockDb, []byte("")}

	server := httptest.NewServer(routers.DefineRoutes(env))
	defer server.Close()

	resp, err := http.Get(server.URL)
	test_util.Ok(t, err)
	test_util.Equals(t, http.StatusOK, resp.StatusCode)
}

func TestAuthenticatedFlow(t *testing.T) {
	mockDb := &MockDataStore{}
	env := &handlers.Environment{mockDb, []byte("")}

	server := httptest.NewServer(routers.DefineRoutes(env))
	defer server.Close()

	// Create testing client
	client := &http.Client{}
	{
		jar, err := cookiejar.New(&cookiejar.Options{})
		test_util.Ok(t, err)

		client.Jar = jar
	}

	// Test login
	userIdAsInt := int64(1)

	t.Run("Login", func(t *testing.T) {
		expectedEmail := "justsomeemail@gmail.com"
		expectedPassword := "worldsBestPassword"

		mockDb.Func_AuthenticateUserCredentials = func(email *models.EmailAddress, password string) error {
			if email.String() == expectedEmail && password == expectedPassword {
				return nil
			}

			return models.CredentialsNotAuthorizedError
		}

		mockDb.Func_GetIdForUserWithEmailAddress = func(email *models.EmailAddress) (models.UserId, error) {
			return models.UserId(userIdAsInt), nil
		}

		userValues := map[string]string{"emailAddress": expectedEmail, "password": expectedPassword}

		userJsonValue, _ := json.Marshal(userValues)

		resp, err := client.Post(server.URL+paths.SessionApi, "application/json", bytes.NewBuffer(userJsonValue))

		test_util.Ok(t, err)
		test_util.Equals(t, http.StatusCreated, resp.StatusCode)

		// TODO make sure that jwt cookie is properly stored.
	})

	// Test Add Note
	noteIdAsInt := int64(33)
	content := "Duuude I just said something cool"

	t.Run("AddNote", func(t *testing.T) {
		mockDb.Func_StoreNewNote = func(note *models.Note) (models.NoteId, error) {
			if note.Content != content {
				return 0, errors.New("Incorrect data")
			}
			return models.NoteId(noteIdAsInt), nil
		}

		noteValues := map[string]string{"content": content}
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
		defer resp.Body.Close()
	})

	// Test get notes
	t.Run("GetNotes", func(t *testing.T) {
		mockDb.Func_GetMyUnpublishedNotes = func(userId models.UserId) (models.NotesById, error) {
			if userIdAsInt != int64(userId) {
				return nil, errors.New("Invalid userId passed in")
			}

			return models.NotesById(map[models.NoteId]*models.Note{
				models.NoteId(noteIdAsInt): &models.Note{
					AuthorId:     models.UserId(userIdAsInt),
					Content:      content,
					CreationTime: time.Now().UTC(),
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
	})

	// Test edit notes
	t.Run("EditNotes", func(t *testing.T) {
		type NoteUpdateForm struct {
			Content string `json:"content"`
		}

		mockDb.Func_GetNoteById = func(models.NoteId) (*models.Note, error) {
			return &models.Note{
				AuthorId:     models.UserId(userIdAsInt),
				Content:      content,
				CreationTime: time.Now().UTC(),
			}, nil
		}

		mockDb.Func_UpdateNoteContent = func(models.NoteId, string) error {
			return nil
		}

		noteForm := &NoteUpdateForm{
			Content: "anything else",
		}

		jsonValue, _ := json.Marshal(noteForm)

		resp, err := sendPutRequest(client, server.URL+paths.NoteApi+"?id="+strconv.FormatInt(noteIdAsInt, 10), "application/json", bytes.NewBuffer(jsonValue))
		test_util.Ok(t, err)
		test_util.Equals(t, http.StatusOK, resp.StatusCode)

	})

	// Test Category
	{
		type CategoryForm struct {
			Category string `json:"category"`
		}

		// Add category
		t.Run("Add Category", func(t *testing.T) {
			metaCategory := models.META

			categoryForm := &CategoryForm{Category: metaCategory.String()}

			mockDb.Func_AssignNoteCategoryRelationship = func(noteId models.NoteId, cat models.NoteCategory) error {
				if int64(noteId) == noteIdAsInt && cat == metaCategory {
					return nil
				}

				return errors.New("Incorrect Data Arrived")
			}

			jsonValue, _ := json.Marshal(categoryForm)

			resp, err := client.Post(server.URL+paths.NoteCategoryApi+"?id="+strconv.FormatInt(noteIdAsInt, 10), "application/json", bytes.NewBuffer(jsonValue))
			test_util.Ok(t, err)
			test_util.Equals(t, http.StatusCreated, resp.StatusCode)

		})

		// Get Cateogry
		t.Run("Get Category", func(t *testing.T) {

			mockDb.Func_GetNoteCategory = func(noteId models.NoteId) (models.NoteCategory, error) {
				if int64(noteId) == noteIdAsInt {
					return models.META, nil
				}

				return 0, errors.New("Incorrect data")
			}

			resp, err := client.Get(server.URL + paths.NoteApi + "?id=" + strconv.FormatInt(noteIdAsInt, 10))
			test_util.Ok(t, err)
			test_util.Equals(t, http.StatusOK, resp.StatusCode)

		})

		// Update cateogry
		t.Run("Update Category", func(t *testing.T) {
			questionCateogry := models.QUESTIONS
			categoryForm := &CategoryForm{Category: questionCateogry.String()}
			jsonValue, _ := json.Marshal(categoryForm)

			mockDb.Func_AssignNoteCategoryRelationship = func(noteId models.NoteId, cat models.NoteCategory) error {
				if int64(noteId) == noteIdAsInt && cat == questionCateogry {
					return nil
				}

				return errors.New("Incorrect Data Arrived")
			}

			resp, err := client.Post(server.URL+paths.NoteCategoryApi+"?id="+strconv.FormatInt(noteIdAsInt, 10), "application/json", bytes.NewBuffer(jsonValue))
			printBody(resp)
			test_util.Ok(t, err)
			test_util.Equals(t, http.StatusCreated, resp.StatusCode)

		})

		// Delete category
		t.Run("Delete Category", func(t *testing.T) {
			mockDb.Func_DeleteNoteCategory = func(noteId models.NoteId) error {
				if int64(noteId) == noteIdAsInt {
					return nil
				}

				return errors.New("Incorrect Data Arrived")
			}

			resp, err := sendDeleteUrl(client, server.URL+paths.NoteCategoryApi+"?id="+strconv.FormatInt(noteIdAsInt, 10))

			test_util.Ok(t, err)
			test_util.Equals(t, http.StatusOK, resp.StatusCode)

		})
	}

	// Test publish notes
	t.Run("Publish Notes", func(t *testing.T) {
		mockDb.Func_PublishNotes = func(userId models.UserId) error {
			return nil
		}
		// publish new api
		resp, err := client.Post(server.URL+paths.PublicationApi, "", nil)
		test_util.Ok(t, err)
		test_util.Equals(t, http.StatusCreated, resp.StatusCode)
	})

	// Delete note
	t.Run("Delete Note", func(t *testing.T) {
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

		resp, err := sendDeleteUrl(client, server.URL+paths.NoteApi+"?id="+strconv.FormatInt(noteIdAsInt, 10))
		test_util.Ok(t, err)

		test_util.Equals(t, http.StatusOK, resp.StatusCode)
	})
}

func sendDeleteRequest(client *http.Client, myUrl string, contentType string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("DELETE", myUrl, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	return client.Do(req)
}
func sendDeleteUrl(client *http.Client, myUrl string) (resp *http.Response, err error) {

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

// Used for debugging tests.
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

type MockDataStore struct {
	Func_StoreNewNote                   func(*models.Note) (models.NoteId, error)
	Func_StoreNewUser                   func(string, *models.EmailAddress, string) error
	Func_AuthenticateUserCredentials    func(*models.EmailAddress, string) error
	Func_GetIdForUserWithEmailAddress   func(*models.EmailAddress) (models.UserId, error)
	Func_GetUsersNotes                  func(models.UserId) (models.NotesById, error)
	Func_DeleteNoteById                 func(models.NoteId) error
	Func_GetMyUnpublishedNotes          func(models.UserId) (models.NotesById, error)
	Func_GetAllUsersById                func() (models.UsersById, error)
	Func_GetAllPublishedNotesVisibleBy  func(models.UserId) (map[int64]models.NotesById, error)
	Func_PublishNotes                   func(models.UserId) error
	Func_StoreNewPublication            func(*models.Publication) (models.PublicationId, error)
	Func_GetNoteById                    func(models.NoteId) (*models.Note, error)
	Func_UpdateNoteContent              func(models.NoteId, string) error
	Func_AssignNoteCategoryRelationship func(models.NoteId, models.NoteCategory) error
	Func_DeleteNoteCategory             func(models.NoteId) error
	Func_GetNoteCategory                func(models.NoteId) (models.NoteCategory, error)
}

func (mock *MockDataStore) StoreNewNote(note *models.Note) (models.NoteId, error) {
	return mock.Func_StoreNewNote(note)
}

func (mock *MockDataStore) StoreNewUser(str1 string, email *models.EmailAddress, str2 string) error {
	return mock.Func_StoreNewUser(str1, email, str2)
}

func (mock *MockDataStore) AuthenticateUserCredentials(email *models.EmailAddress, str string) error {
	return mock.Func_AuthenticateUserCredentials(email, str)
}

func (mock *MockDataStore) GetIdForUserWithEmailAddress(email *models.EmailAddress) (models.UserId, error) {
	return mock.Func_GetIdForUserWithEmailAddress(email)
}

func (mock *MockDataStore) GetUsersNotes(userId models.UserId) (models.NotesById, error) {
	return mock.Func_GetUsersNotes(userId)
}

func (mock *MockDataStore) DeleteNoteById(noteId models.NoteId) error {
	return mock.Func_DeleteNoteById(noteId)
}

func (mock *MockDataStore) GetMyUnpublishedNotes(userId models.UserId) (models.NotesById, error) {
	return mock.Func_GetMyUnpublishedNotes(userId)
}

func (mock *MockDataStore) GetAllUsersById() (models.UsersById, error) {
	return mock.Func_GetAllUsersById()
}

func (mock *MockDataStore) GetAllPublishedNotesVisibleBy(userId models.UserId) (map[int64]models.NotesById, error) {
	return mock.Func_GetAllPublishedNotesVisibleBy(userId)
}

func (mock *MockDataStore) PublishNotes(userId models.UserId) error {
	return mock.Func_PublishNotes(userId)
}

func (mock *MockDataStore) StoreNewPublication(publication *models.Publication) (models.PublicationId, error) {
	return mock.Func_StoreNewPublication(publication)
}

func (mock *MockDataStore) GetNoteById(noteId models.NoteId) (*models.Note, error) {
	return mock.Func_GetNoteById(noteId)
}

func (mock *MockDataStore) UpdateNoteContent(noteId models.NoteId, content string) error {
	return mock.Func_UpdateNoteContent(noteId, content)
}

func (mock *MockDataStore) GetNoteCategory(noteId models.NoteId) (models.NoteCategory, error) {
	return mock.Func_GetNoteCategory(noteId)
}

func (mock *MockDataStore) AssignNoteCategoryRelationship(noteId models.NoteId, category models.NoteCategory) error {
	return mock.Func_AssignNoteCategoryRelationship(noteId, category)
}
func (mock *MockDataStore) DeleteNoteCategory(noteId models.NoteId) error {
	return mock.Func_DeleteNoteCategory(noteId)
}
