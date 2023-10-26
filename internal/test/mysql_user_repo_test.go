package test

import (
	"database/sql"
	"fmt"
	"github.com/vlasdash/redditclone/internal/user"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"reflect"
	"testing"
)

func TestUserGetByIDCorrect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Logf("cant create mock: %v", err)
		return
	}
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}
	}()

	var userID uint = 1
	rows := sqlmock.NewRows([]string{"id", "username", "password"})
	expected := []*user.User{
		{userID, "username", "password"},
	}
	for _, u := range expected {
		rows = rows.AddRow(u.ID, u.Username, u.Password)
	}

	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE id = ?").
		WithArgs(userID).
		WillReturnRows(rows)

	repo := &user.MySQLRepo{
		DB: db,
	}
	u, err := repo.GetByID(userID)
	if err != nil {
		t.Errorf("wrong result, got error: %v", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectation error: %v", err)
		return
	}
	if !reflect.DeepEqual(u, expected[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", expected[0], u)
	}
}

func TestUserGetByIDError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Logf("cant create mock: %v", err)
		return
	}
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}
	}()

	var userID uint = 1
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE id = ?").
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	repo := &user.MySQLRepo{
		DB: db,
	}
	_, err = repo.GetByID(userID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectation error: %v", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if err != user.ErrNoExist {
		t.Errorf("expected error %v, got error %v", sql.ErrNoRows, err)
		return
	}

	rows := sqlmock.
		NewRows([]string{"id", "username"}).
		AddRow(1, "username")

	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE id = ?").
		WithArgs(userID).
		WillReturnRows(rows)

	_, err = repo.GetByID(userID)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectation error: %v", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestUserGetByUsernameCorrect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Logf("cant create mock: %v", err)
		return
	}
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}
	}()

	username := "username"
	rows := sqlmock.NewRows([]string{"id", "username", "password"})
	expected := []*user.User{
		{1, username, "password"},
	}
	for _, u := range expected {
		rows = rows.AddRow(u.ID, u.Username, u.Password)
	}

	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnRows(rows)

	repo := &user.MySQLRepo{
		DB: db,
	}
	u, err := repo.GetByUsername(username)
	if err != nil {
		t.Errorf("wrong result, got error: %v", err)
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectation error: %v", err)
		return
	}
	if !reflect.DeepEqual(u, expected[0]) {
		t.Errorf("wrong result, expected %#v, got %#v", expected[0], u)
	}
}

func TestUserGetByUsernameError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Logf("cant create mock: %v", err)
		return
	}
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}
	}()

	username := "username"
	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnError(sql.ErrNoRows)

	repo := &user.MySQLRepo{
		DB: db,
	}
	_, err = repo.GetByUsername(username)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectation error: %v", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if err != user.ErrNoExist {
		t.Errorf("expected error %v, got error %v", sql.ErrNoRows, err)
		return
	}

	rows := sqlmock.
		NewRows([]string{"id", "username"}).
		AddRow(1, "username")

	mock.
		ExpectQuery("SELECT id, username, password FROM users WHERE username = ?").
		WithArgs(username).
		WillReturnRows(rows)

	_, err = repo.GetByUsername(username)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectation error: %v", err)
		return
	}
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestUserCreateCorrect(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Logf("cant create mock: %v", err)
		return
	}
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}
	}()

	repo := &user.MySQLRepo{
		DB: db,
	}

	username := "username"
	password := "password"
	var expectedID uint = 1
	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password).
		WillReturnResult(sqlmock.NewResult(1, 1))

	id, err := repo.Create(username, password)
	if err != nil {
		t.Errorf("wrong result, got error: %v", err)
		return
	}
	if id != expectedID {
		t.Errorf("wrong result, expected %#v, got %#v", expectedID, id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUserCreateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Logf("cant create mock: %v", err)
		return
	}
	defer func() {
		err := db.Close()
		if err != nil {
			return
		}
	}()

	repo := &user.MySQLRepo{
		DB: db,
	}

	username := "username"
	password := "password"
	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password).
		WillReturnError(fmt.Errorf("something went wrong"))

	_, err = repo.Create(username, password)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}

	mock.
		ExpectExec("INSERT INTO users").
		WithArgs(username, password).
		WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("bad insertion")))

	_, err = repo.Create(username, password)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
		return
	}
}
