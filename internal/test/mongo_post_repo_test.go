package test

import (
	"fmt"
	"github.com/vlasdash/redditclone/internal/post"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"reflect"
	"testing"
)

func TestPostGetAll(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		id := primitive.NewObjectID()
		expectedPosts := []*post.Post{
			{
				ID:             id.Hex(),
				Category:       "music",
				CreateDate:     "date",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		}
		startCursor := mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", expectedPosts[0].Category},
			{"create_date", expectedPosts[0].CreateDate},
			{"text", expectedPosts[0].Text},
			{"title", expectedPosts[0].Title},
			{"type", expectedPosts[0].Type},
			{"views", expectedPosts[0].Views},
			{"votes", expectedPosts[0].Votes},
			{"comment_ids", expectedPosts[0].CommentIDs},
			{"author_id", expectedPosts[0].AuthorID},
			{"upvotes_count", expectedPosts[0].UpvotesCount},
			{"downvotes_count", expectedPosts[0].DownvotesCount},
		})
		endCursor := mtest.CreateCursorResponse(0, "reddit.posts", mtest.NextBatch)
		mt.AddMockResponses(startCursor, endCursor)

		posts, err := postRepo.GetAll()
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
		if !reflect.DeepEqual(posts, expectedPosts) {
			t.Errorf("wrong result, expected %#v, got %#v", expectedPosts, posts)
		}
	})

	mt.Run("find error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})

		_, err := postRepo.GetAll()
		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		_, err := postRepo.GetAll()

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestPostCreate(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		post := &post.Post{
			Category: "music",
			Text:     "text",
			Title:    "title",
			Type:     "link",
			AuthorID: 1,
		}

		mt.AddMockResponses(mtest.CreateSuccessResponse())

		_, err := postRepo.Create(post)
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
	})

	mt.Run("error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		post := &post.Post{
			Category: "music",
			Text:     "text",
			Title:    "title",
			Type:     "link",
			AuthorID: 1,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})

		_, err := postRepo.Create(post)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err)
			return
		}
	})
}

func TestPostGetByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		id := primitive.NewObjectID()
		expectedPost := &post.Post{
			ID:             id.Hex(),
			Category:       "music",
			CreateDate:     "date",
			Text:           "text",
			Title:          "title",
			Type:           "link",
			Views:          0,
			Votes:          make([]*post.Vote, 0),
			CommentIDs:     make([]string, 0),
			AuthorID:       1,
			UpvotesCount:   0,
			DownvotesCount: 0,
		}

		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"value", bson.D{
				{"_id", id},
				{"category", expectedPost.Category},
				{"create_date", expectedPost.CreateDate},
				{"text", expectedPost.Text},
				{"title", expectedPost.Title},
				{"type", expectedPost.Type},
				{"views", expectedPost.Views},
				{"votes", expectedPost.Votes},
				{"comment_ids", expectedPost.CommentIDs},
				{"author_id", expectedPost.AuthorID},
				{"upvotes_count", expectedPost.UpvotesCount},
				{"downvotes_count", expectedPost.DownvotesCount},
			}}})

		post, err := postRepo.GetByID(expectedPost.ID, 0)
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
		if !reflect.DeepEqual(post, expectedPost) {
			t.Errorf("wrong result, expected %#v, got %#v", expectedPost, post)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
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

		_, err := postRepo.GetByID(id.Hex(), 0)
		if err != post.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrNotExist, err)
			return
		}
	})

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		_, err := postRepo.GetByID("bad_id", 0)
		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"value", bson.D{
				{"_id", "notObjectID"},
			}}})

		_, err := postRepo.GetByID(id.Hex(), 0)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestPostGetByCategory(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		id := primitive.NewObjectID()
		expectedPosts := []*post.Post{
			{
				ID:             id.Hex(),
				Category:       "music",
				CreateDate:     "date",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		}
		startCursor := mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", expectedPosts[0].Category},
			{"create_date", expectedPosts[0].CreateDate},
			{"text", expectedPosts[0].Text},
			{"title", expectedPosts[0].Title},
			{"type", expectedPosts[0].Type},
			{"views", expectedPosts[0].Views},
			{"votes", expectedPosts[0].Votes},
			{"comment_ids", expectedPosts[0].CommentIDs},
			{"author_id", expectedPosts[0].AuthorID},
			{"upvotes_count", expectedPosts[0].UpvotesCount},
			{"downvotes_count", expectedPosts[0].DownvotesCount},
		})
		endCursor := mtest.CreateCursorResponse(0, "reddit.posts", mtest.NextBatch)
		mt.AddMockResponses(startCursor, endCursor)

		posts, err := postRepo.GetByCategory("music")
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
		if !reflect.DeepEqual(posts, expectedPosts) {
			t.Errorf("wrong result, expected %#v, got %#v", expectedPosts, posts)
		}
	})

	mt.Run("find error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})

		_, err := postRepo.GetByCategory("music")

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		_, err := postRepo.GetByCategory("music")

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestPostGetByAuthor(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		id := primitive.NewObjectID()
		expectedPosts := []*post.Post{
			{
				ID:             id.Hex(),
				Category:       "music",
				CreateDate:     "date",
				Text:           "text",
				Title:          "title",
				Type:           "link",
				Views:          0,
				Votes:          make([]*post.Vote, 0),
				CommentIDs:     make([]string, 0),
				AuthorID:       1,
				UpvotesCount:   0,
				DownvotesCount: 0,
			},
		}
		startCursor := mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", expectedPosts[0].Category},
			{"create_date", expectedPosts[0].CreateDate},
			{"text", expectedPosts[0].Text},
			{"title", expectedPosts[0].Title},
			{"type", expectedPosts[0].Type},
			{"views", expectedPosts[0].Views},
			{"votes", expectedPosts[0].Votes},
			{"comment_ids", expectedPosts[0].CommentIDs},
			{"author_id", expectedPosts[0].AuthorID},
			{"upvotes_count", expectedPosts[0].UpvotesCount},
			{"downvotes_count", expectedPosts[0].DownvotesCount},
		})
		endCursor := mtest.CreateCursorResponse(0, "reddit.posts", mtest.NextBatch)
		mt.AddMockResponses(startCursor, endCursor)

		posts, err := postRepo.GetByAuthor(1)
		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
		if !reflect.DeepEqual(posts, expectedPosts) {
			t.Errorf("wrong result, expected %#v, got %#v", expectedPosts, posts)
		}
	})

	mt.Run("find error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})

		_, err := postRepo.GetByAuthor(1)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		_, err := postRepo.GetByAuthor(1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestPostAddComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"value", bson.D{
				{"_id", id},
				{"category", "music"},
				{"create_date", "date"},
				{"text", "text"},
				{"title", "title"},
				{"type", "link"},
				{"views", 0},
				{"votes", make([]*post.Vote, 0)},
				{"comment_ids", make([]string, 0)},
				{"author_id", 1},
				{"upvotes_count", 0},
				{"downvotes_count", 0},
			}}})

		err := postRepo.AddComment(id.Hex(), "comment_id")

		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
	})

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		err := postRepo.AddComment("bad_id", "comment_id")
		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("update error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})
		id := primitive.NewObjectID()

		err := postRepo.AddComment(id.Hex(), "comment_id")

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{{"ok", 1}})

		err := postRepo.AddComment(id.Hex(), "comment_id")

		if err != post.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrNotExist, err)
			return
		}
	})
}

func TestPostUpvote(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		err := postRepo.Upvote("bad_id", 1)
		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("find and update error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})
		id := primitive.NewObjectID()

		err := postRepo.Upvote(id.Hex(), 1)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("update command error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedError := "no responses remaining"

		mt.AddMockResponses(bson.D{{"ok", 1}})

		err := postRepo.Upvote(id.Hex(), 1)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})
}

func TestPostDownvote(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		err := postRepo.Downvote("bad_id", 1)
		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("find and update error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})
		id := primitive.NewObjectID()

		err := postRepo.Downvote(id.Hex(), 1)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("update command error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedError := "no responses remaining"

		mt.AddMockResponses(bson.D{{"ok", 1}})

		err := postRepo.Downvote(id.Hex(), 1)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})
}

func TestPostUnvote(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		err := postRepo.Unvote("bad_id", 1)
		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("find one error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})
		id := primitive.NewObjectID()

		err := postRepo.Unvote(id.Hex(), 1)

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		res := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", "reddit", "posts"),
			mtest.FirstBatch)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", "reddit", "posts"),
			mtest.NextBatch)
		mt.AddMockResponses(res, end)
		id := primitive.NewObjectID()

		err := postRepo.Unvote(id.Hex(), 1)
		if err != post.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrNotExist, err)
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		err := postRepo.Unvote(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})

	mt.Run("like value", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "no responses remaining"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", "music"},
			{"create_date", "date"},
			{"text", "text"},
			{"title", "title"},
			{"type", "link"},
			{"views", 0},
			{"votes", []*post.Vote{
				{
					UserID: 1,
					Value:  post.Like,
				},
			},
			},
			{"comment_ids", make([]string, 0)},
			{"author_id", 1},
			{"upvotes_count", 0},
			{"downvotes_count", 0},
		}))

		err := postRepo.Unvote(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})

	mt.Run("unlike value", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "no responses remaining"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", "music"},
			{"create_date", "date"},
			{"text", "text"},
			{"title", "title"},
			{"type", "link"},
			{"views", 0},
			{"votes", []*post.Vote{
				{
					UserID: 1,
					Value:  post.Unlike,
				},
			},
			},
			{"comment_ids", make([]string, 0)},
			{"author_id", 1},
			{"upvotes_count", 0},
			{"downvotes_count", 0},
		}))

		err := postRepo.Unvote(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestPostDelete(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		err := postRepo.Delete("bad_id", 1)

		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		res := mtest.CreateCursorResponse(
			1,
			fmt.Sprintf("%s.%s", "reddit", "posts"),
			mtest.FirstBatch)
		end := mtest.CreateCursorResponse(
			0,
			fmt.Sprintf("%s.%s", "reddit", "posts"),
			mtest.NextBatch)
		mt.AddMockResponses(res, end)
		id := primitive.NewObjectID()

		err := postRepo.Delete(id.Hex(), 1)

		if err != post.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrNotExist, err)
			return
		}
	})

	mt.Run("decode error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()
		expectedErr := "error decoding key _id: an ObjectID string must be exactly 12 bytes long (got 11)"

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{{"_id", "notObjectID"}}))

		err := postRepo.Delete(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})

	mt.Run("error access", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		id := primitive.NewObjectID()
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.posts", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", "music"},
			{"create_date", "date"},
			{"text", "text"},
			{"title", "title"},
			{"type", "link"},
			{"views", 0},
			{"votes", make([]*post.Vote, 0)},
			{"comment_ids", make([]string, 0)},
			{"author_id", 2},
			{"upvotes_count", 0},
			{"downvotes_count", 0},
		}))

		err := postRepo.Delete(id.Hex(), 1)

		if err != post.ErrNoAccess {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrNoAccess, err)
			return
		}
	})

	mt.Run("error delete one", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedErr := "no responses remaining"
		id := primitive.NewObjectID()

		mt.AddMockResponses(mtest.CreateCursorResponse(1, "reddit.comments", mtest.FirstBatch, bson.D{
			{"_id", id},
			{"category", "music"},
			{"create_date", "date"},
			{"text", "text"},
			{"title", "title"},
			{"type", "link"},
			{"views", 0},
			{"votes", make([]*post.Vote, 0)},
			{"comment_ids", make([]string, 0)},
			{"author_id", 1},
			{"upvotes_count", 0},
			{"downvotes_count", 0},
		}))

		err := postRepo.Delete(id.Hex(), 1)

		if err.Error() != expectedErr {
			t.Errorf("wrong result, expected error %v, got %v", expectedErr, err)
			return
		}
	})
}

func TestPostDeleteComment(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"value", bson.D{
				{"_id", id},
				{"category", "music"},
				{"create_date", "date"},
				{"text", "text"},
				{"title", "title"},
				{"type", "link"},
				{"views", 0},
				{"votes", make([]*post.Vote, 0)},
				{"comment_ids", make([]string, 0)},
				{"author_id", 1},
				{"upvotes_count", 0},
				{"downvotes_count", 0},
			}}})

		err := postRepo.DeleteComment(id.Hex(), "comment_id")

		if err != nil {
			t.Errorf("wrong result, got error: %v", err)
			return
		}
	})

	mt.Run("bad id", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}

		err := postRepo.DeleteComment("bad_id", "comment_id")
		if err != post.ErrInvalidID {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrInvalidID, err)
			return
		}
	})

	mt.Run("update error", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		expectedError := "command failed"

		mt.AddMockResponses(bson.D{{"ok", 0}})
		id := primitive.NewObjectID()

		err := postRepo.DeleteComment(id.Hex(), "comment_id")

		if err.Error() != expectedError {
			t.Errorf("wrong result, expected error %v, got %v", expectedError, err.Error())
			return
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		collection := mt.Coll
		postRepo := post.MongoRepo{
			Posts: collection,
		}
		id := primitive.NewObjectID()

		mt.AddMockResponses(bson.D{{"ok", 1}})

		err := postRepo.DeleteComment(id.Hex(), "comment_id")

		if err != post.ErrNotExist {
			t.Errorf("wrong result, expected error %v, got %v", post.ErrNotExist, err)
			return
		}
	})
}
