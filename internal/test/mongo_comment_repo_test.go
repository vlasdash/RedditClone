package test

import (
	"fmt"
	"github.com/vlasdash/redditclone/internal/comment"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
	"time"
)

func TestCommentGetByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		id := primitive.NewObjectID()
		expectedComment := comment.Comment{
			ID:         id.Hex(),
			AuthorID:   1,
			CreateDate: time.Now().Format(time.RFC3339),
			Body:       "body",
		}

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"author_id", expectedComment.AuthorID},
			{"create_date", expectedComment.CreateDate},
			{"body", expectedComment.Body},
		}))

		comment, err := commentRepo.GetByID(expectedComment.ID)
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
		if !reflect.DeepEqual(comment, &expectedComment) {
			t.Errorf("wrong result, expected %#v, got %#v", &expectedComment, comment)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		res := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", "reddit", "comments"),
			mtest.FirstBatch)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", "reddit", "comments"),
			mtest.NextBatch)
		mt.AddMockResponses(res, end)
		id := primitive.NewObjectID()

		_, err := commentRepo.GetByID(id.Hex())
		if err != comment.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", comment.ErrNotExist, err)
			return
		}
	})

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		_, err := commentRepo.GetByID("bad_id")
		if err.Error() != comment.ErrInvalidID.Error() {
			t.Errorf("wrong result, expected error %v, got %v", comment.ErrInvalidID, err)
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		_, err := commentRepo.GetByID(id.Hex())

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestCommentAdd(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		_, err := commentRepo.Add(1, "body")
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
	})

	mt.Run("error", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   1,
			Code:    11000,
			Message: "duplicate key error",
		}))

		_, err := commentRepo.Add(1, "body")

		if !mongo.IsDuplicateKeyError(err) {
			t.Errorf("wrong result, expected error mongo.DuplicateKeyError, got %v", err)
			return
		}
	})
}

func TestCommentDelete(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		err := commentRepo.Delete("bad_id", 1)

		if err.Error() != comment.ErrInvalidID.Error() {
			t.Errorf("wrong result, expected error %v, got %v", comment.ErrInvalidID, err)
			return
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		res := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", "reddit", "comments"),
			mtest.FirstBatch)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", "reddit", "comments"),
			mtest.NextBatch)
		mt.AddMockResponses(res, end)
		id := primitive.NewObjectID()

		err := commentRepo.Delete(id.Hex(), 1)

		if err != comment.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", comment.ErrNotExist, err)
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		err := commentRepo.Delete(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})

	mt.Run("error access", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}

		id := primitive.NewObjectID()
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{
			{"_id", id.Hex()},
			{"author_id", 2},
			{"create_date", "date"},
			{"body", "body"},
		}))

		err := commentRepo.Delete(id.Hex(), 1)

		if err != comment.ErrNoAccess {
			t.Errorf("wrong result, expected error %v, got %v", comment.ErrNoAccess, err)
			return
		}
	})

	mt.Run("error delete one", func(mt *mtest.T) {
		collection := mt.Coll
		commentRepo := comment.MongoRepo{
			Comments: collection,
		}
		id := primitive.NewObjectID()

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{
			{"_id", id.Hex()},
			{"author_id", 1},
			{"create_date", "date"},
			{"body", "body"},
		}))
		expectedErr := "no responses remaining"

		err := commentRepo.Delete(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}
