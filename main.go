package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Booking struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Room  string             `bson:"room" json:"room"`
	Start time.Time          `bson:"start" json:"start"`
	End   time.Time          `bson:"end" json:"end"`
}

func main() {
	ctx := context.Background()
	//client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017/"))
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("DATABASE_URL")))

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	//coll := client.Database("booking").Collection("booking")
	coll := client.Database(os.Getenv("DATABASE_NAME")).Collection("booking")

	r := gin.Default()

	r.POST("/bookings", wrapError(coll, createHandler))
	r.GET("/bookings", wrapError(coll, listBookingHandler))
	r.GET("/bookings/:id", wrapError(coll, getBookingHandler))
	r.DELETE("/bookings/:id", wrapError(coll, deleteBookingHandler))
	r.Run(":8000")
}

func wrapError(coll *mongo.Collection, h func(context.Context, *gin.Context, *mongo.Collection) error) func(*gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		err := h(ctx, c, coll)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithError(http.StatusNotFound, err)
			} else {
				c.AbortWithError(http.StatusInternalServerError, err)
			}
		}
	}
}

func newBooking() *Booking {
	return &Booking{}
}

func createHandler(ctx context.Context, c *gin.Context, coll *mongo.Collection) error {
	todo, err := createBooking(ctx, c, coll)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, todo)
	return nil
}

func createBooking(ctx context.Context, c *gin.Context, coll *mongo.Collection) (*Booking, error) {
	book := newBooking()
	if err := c.Bind(&book); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return nil, err
	}
	result, err := coll.InsertOne(ctx, book)
	if err != nil {
		return nil, err
	}
	book.ID = result.InsertedID.(primitive.ObjectID)
	return book, nil
}

func listBookingHandler(ctx context.Context, c *gin.Context, coll *mongo.Collection) error {
	todos, err := listBookings(ctx, coll)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, todos)
	return nil
}

func listBookings(ctx context.Context, coll *mongo.Collection) ([]*Booking, error) {
	cur, err := coll.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	bookingList := []*Booking{}

	for cur.Next(ctx) {
		book := newBooking()
		if err := cur.Decode(book); err != nil {
			return nil, err
		}
		bookingList = append(bookingList, book)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(bookingList, func(i, j int) bool { return bookingList[i].Start.Before(bookingList[j].Start) })
	return bookingList, nil
}

func getBookingHandler(ctx context.Context, c *gin.Context, coll *mongo.Collection) error {
	book, err := getBooking(ctx, c, coll)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, book)
	return nil
}

func getBooking(ctx context.Context, c *gin.Context, coll *mongo.Collection) (*Booking, error) {
	id, _ := primitive.ObjectIDFromHex(c.Param("id"))
	book := newBooking()
	err := coll.FindOne(ctx, bson.D{{"_id", id}}).Decode(&book)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.AbortWithError(http.StatusNotFound, err)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return book, nil
	}
	return book, nil
}

func deleteBookingHandler(ctx context.Context, c *gin.Context, coll *mongo.Collection) error {
	err := deleteBooking(ctx, c, coll)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, "")
	return nil
}

func deleteBooking(ctx context.Context, c *gin.Context, coll *mongo.Collection) error {
	id, _ := primitive.ObjectIDFromHex(c.Param("id"))
	_, err := coll.DeleteOne(c.Request.Context(), bson.D{{"_id", id}})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.AbortWithError(http.StatusNotFound, err)
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
		}
		return err
	}
	return err
}
