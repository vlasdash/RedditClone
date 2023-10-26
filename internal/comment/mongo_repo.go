package comment

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type MongoRepo struct {
	Comments *mongo.Collection
	DB       *mongo.Database
}

var _ CommentRepo = (*MongoRepo)(nil)

type Item struct {
	ID         primitive.ObjectID `bson:"_id"`
	AuthorID   uint               `bson:"author_id"`
	CreateDate string             `bson:"create_date"`
	Body       string             `bson:"body"`
}

func NewMongoRepo(db *mongo.Database) *MongoRepo {
	comments := db.Collection("comments")

	return &MongoRepo{
		Comments: comments,
		DB:       db,
	}
}

func (r *MongoRepo) Add(userID uint, body string) (id string, err error) {
	comment := Item{
		ID:         primitive.NewObjectID(),
		AuthorID:   userID,
		CreateDate: time.Now().Format(time.RFC3339),
		Body:       body,
	}

	_, err = r.Comments.InsertOne(context.TODO(), comment)
	if err != nil {
		return "", err
	}

	return comment.ID.Hex(), nil
}

func (r *MongoRepo) GetByID(id string) (*Comment, error) {
	item := &Item{}
	itemID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}
	filter := bson.M{"_id": itemID}

	err = r.Comments.FindOne(context.TODO(), filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrNotExist
		}

		return nil, err
	}

	comment := &Comment{
		ID:         item.ID.Hex(),
		AuthorID:   item.AuthorID,
		CreateDate: item.CreateDate,
		Body:       item.Body,
	}

	return comment, nil
}

func (r *MongoRepo) Delete(id string, userID uint) error {
	item := &Item{}
	itemID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}
	filter := bson.M{"_id": itemID}

	err = r.Comments.FindOne(context.TODO(), filter).Decode(&item)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrNotExist
		}

		return err
	}

	if item.AuthorID != userID {
		return ErrNoAccess
	}

	_, err = r.Comments.DeleteOne(context.TODO(), filter)
	return err
}
