package post

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
	"time"
)

type Item struct {
	ID             primitive.ObjectID `bson:"_id"`
	Category       string             `bson:"category"`
	CreateDate     string             `bson:"create_date"`
	Text           string             `bson:"text"`
	URL            string             `bson:"url"`
	Title          string             `bson:"title"`
	Type           string             `bson:"type"`
	Views          int                `bson:"views"`
	Votes          []*Vote            `bson:"votes"`
	CommentIDs     []string           `bson:"comment_ids"`
	AuthorID       uint               `bson:"author_id"`
	UpvotesCount   int                `bson:"upvotes_count"`
	DownvotesCount int                `bson:"downvotes_count"`
}

type MongoRepo struct {
	Posts *mongo.Collection
	DB    *mongo.Database
}

var _ PostRepo = (*MongoRepo)(nil)

func NewMongoRepo(db *mongo.Database) *MongoRepo {
	collection := db.Collection("posts")

	return &MongoRepo{
		Posts: collection,
		DB:    db,
	}
}

func (r *MongoRepo) GetAll() ([]*Post, error) {
	var items []*Item

	cursor, err := r.Posts.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	posts := make([]*Post, 0, len(items))
	for _, item := range items {
		posts = append(posts, &Post{
			ID:             item.ID.Hex(),
			Category:       item.Category,
			CreateDate:     item.CreateDate,
			Text:           item.Text,
			URL:            item.URL,
			Title:          item.Title,
			Type:           item.Type,
			Views:          item.Views,
			Votes:          item.Votes,
			CommentIDs:     item.CommentIDs,
			AuthorID:       item.AuthorID,
			UpvotesCount:   item.UpvotesCount,
			DownvotesCount: item.DownvotesCount,
		})
	}

	return posts, nil

}

func (r *MongoRepo) Create(post *Post) (id string, err error) {
	item := Item{
		ID:             primitive.NewObjectID(),
		Category:       post.Category,
		CreateDate:     time.Now().Format(time.RFC3339),
		Text:           post.Text,
		URL:            post.URL,
		Title:          post.Title,
		Type:           post.Type,
		Views:          0,
		AuthorID:       post.AuthorID,
		UpvotesCount:   1,
		DownvotesCount: 0,
		Votes: []*Vote{
			{
				UserID: post.AuthorID,
				Value:  Like,
			},
		},
	}
	item.CommentIDs = make([]string, 0)

	_, err = r.Posts.InsertOne(context.TODO(), item)
	if err != nil {
		return "", err
	}

	return item.ID.Hex(), nil
}

func (r *MongoRepo) GetByID(id string, viewsUpdate int) (*Post, error) {
	itemID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	filter := bson.M{"_id": itemID}
	update := bson.M{
		"$inc": bson.M{
			"views": viewsUpdate,
		},
	}
	item := &Item{}
	err = r.Posts.FindOneAndUpdate(context.TODO(), filter, update).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotExist
		}

		return nil, err
	}

	post := &Post{
		ID:             item.ID.Hex(),
		Category:       item.Category,
		CreateDate:     item.CreateDate,
		Text:           item.Text,
		URL:            item.URL,
		Title:          item.Title,
		Type:           item.Type,
		Views:          item.Views + viewsUpdate,
		Votes:          item.Votes,
		CommentIDs:     item.CommentIDs,
		AuthorID:       item.AuthorID,
		UpvotesCount:   item.UpvotesCount,
		DownvotesCount: item.DownvotesCount,
	}

	return post, nil
}

func (r *MongoRepo) GetByCategory(category string) ([]*Post, error) {
	filter := bson.M{"category": category}
	cursor, err := r.Posts.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	var items []*Item
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	posts := make([]*Post, 0, len(items))
	for _, item := range items {
		post := &Post{
			ID:             item.ID.Hex(),
			Category:       item.Category,
			CreateDate:     item.CreateDate,
			Text:           item.Text,
			URL:            item.URL,
			Title:          item.Title,
			Type:           item.Type,
			Views:          item.Views,
			Votes:          item.Votes,
			CommentIDs:     item.CommentIDs,
			AuthorID:       item.AuthorID,
			UpvotesCount:   item.UpvotesCount,
			DownvotesCount: item.DownvotesCount,
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (r *MongoRepo) GetByAuthor(id uint) ([]*Post, error) {
	filter := bson.M{"author_id": id}
	cursor, err := r.Posts.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	var items []*Item
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	posts := make([]*Post, 0, len(items))
	for _, item := range items {
		post := &Post{
			ID:             item.ID.Hex(),
			Category:       item.Category,
			CreateDate:     item.CreateDate,
			Text:           item.Text,
			URL:            item.URL,
			Title:          item.Title,
			Type:           item.Type,
			Views:          item.Views,
			Votes:          item.Votes,
			CommentIDs:     item.CommentIDs,
			AuthorID:       item.AuthorID,
			UpvotesCount:   item.UpvotesCount,
			DownvotesCount: item.DownvotesCount,
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (r *MongoRepo) AddComment(postID string, commentID string) error {
	itemID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return ErrInvalidID
	}

	update := bson.M{"$push": bson.M{"comment_ids": commentID}}

	res := r.Posts.FindOneAndUpdate(context.TODO(), bson.M{"_id": itemID}, update)
	if res.Err() == mongo.ErrNoDocuments {
		return ErrNotExist
	}

	return res.Err()
}

func (r *MongoRepo) Upvote(postID string, voter uint) error {
	itemID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return ErrInvalidID
	}

	update := bson.M{
		"$set": bson.M{
			"votes.$.value": Like,
		},
		"$inc": bson.M{
			"upvotes_count":   1,
			"downvotes_count": -1,
		},
	}
	filter := bson.M{"_id": itemID, "votes.user_id": voter}

	res := r.Posts.FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != mongo.ErrNoDocuments {
		return res.Err()
	}

	update = bson.M{
		"$push": bson.M{
			"votes": Vote{
				UserID: voter,
				Value:  Like,
			},
		},
		"$inc": bson.M{
			"upvotes_count": 1,
		},
	}
	option := options.Update().SetUpsert(true)

	_, err = r.Posts.UpdateByID(context.TODO(), itemID, update, option)
	return err
}

func (r *MongoRepo) Downvote(postID string, voter uint) error {
	itemID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return ErrInvalidID
	}

	filter := bson.M{"_id": itemID, "votes.user_id": voter}
	update := bson.M{
		"$set": bson.M{
			"votes.$.value": Unlike,
		},
		"$inc": bson.M{
			"upvotes_count":   -1,
			"downvotes_count": 1,
		},
	}

	res := r.Posts.FindOneAndUpdate(context.TODO(), filter, update)
	if res.Err() != mongo.ErrNoDocuments {
		return res.Err()
	}

	update = bson.M{
		"$push": bson.M{
			"votes": Vote{
				UserID: voter,
				Value:  Unlike,
			},
		},
		"$inc": bson.M{
			"downvotes_count": 1,
		},
	}
	option := options.Update().SetUpsert(true)

	_, err = r.Posts.UpdateByID(context.TODO(), itemID, update, option)
	return err
}

func (r *MongoRepo) Unvote(postID string, voter uint) error {
	itemID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return ErrInvalidID
	}

	filter := bson.M{"_id": itemID, "votes.user_id": voter}
	item := &Item{}
	err = r.Posts.FindOne(context.TODO(), filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrNotExist
		}

		return err
	}

	update := bson.M{}
	if item.Votes[0].Value == Like {
		update = bson.M{
			"$pull": bson.M{
				"votes": bson.M{"user_id": voter},
			},
			"$inc": bson.M{
				"upvotes_count": -1,
			},
		}
	} else {
		update = bson.M{
			"$pull": bson.M{
				"votes": bson.M{"user_id": voter},
			},
			"$inc": bson.M{
				"downvotes_count": -1,
			},
		}
	}
	option := options.Update().SetUpsert(true)
	_, err = r.Posts.UpdateByID(context.TODO(), itemID, update, option)

	return err
}

func (r *MongoRepo) Delete(postID string, userID uint) error {
	itemID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return ErrInvalidID
	}
	filter := bson.M{"_id": itemID}
	item := &Item{}

	err = r.Posts.FindOne(context.TODO(), filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrNotExist
		}

		return err
	}

	if item.AuthorID != userID {
		return ErrNoAccess
	}

	_, err = r.Posts.DeleteOne(context.TODO(), filter)

	return err
}

func (r *MongoRepo) DeleteComment(postID string, commentID string) error {
	itemID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return ErrInvalidID
	}

	update := bson.M{"$pull": bson.M{"comment_ids": commentID}}

	res := r.Posts.FindOneAndUpdate(context.TODO(), bson.M{"_id": itemID}, update)
	if res.Err() == mongo.ErrNoDocuments {
		return ErrNotExist
	}

	return res.Err()
}
