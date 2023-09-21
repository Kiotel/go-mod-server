package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"strconv"
	"time"
)

type Mod struct {
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	Image       string    `json:"image" bson:"image"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		Level(zerolog.TraceLevel).
		With().
		Timestamp().
		Caller().
		Int("pid", os.Getpid()).
		Logger()

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://127.0.0.1:27017/"))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	log.Info().Msg("ping")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		panic(err)
	}
	log.Info().Msg("pong")

	modsCollection := client.Database("suxen").Collection("mods")

	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})

	app.Use(cors.New())

	app.Get("/accd", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "abed",
		})
	})
	app.Get("/mods", func(c *fiber.Ctx) error {
		page, err := strconv.Atoi(c.Query("page", "-1"))
		var opts *options.FindOptions
		if page != -1 {
			opts = options.Find().SetLimit(int64(page * 20)).SetSkip(int64(page*20 - 20))
		}

		cursor, err := modsCollection.Find(context.TODO(), bson.D{}, opts)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve documents",
			})
		}
		defer cursor.Close(context.TODO())

		var mods []bson.M
		if err := cursor.All(context.TODO(), &mods); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to decode documents",
			})
		}

		return c.Status(fiber.StatusOK).JSON(mods)
	})

	app.Put("/create", func(c *fiber.Ctx) error {
		mod := new(Mod)
		mod.UpdatedAt = time.Now() // mongo doesn't autofill time

		if err := c.BodyParser(mod); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Body fields are missing",
			})
		}

		var foundMod Mod
		if err := modsCollection.FindOne(context.TODO(), bson.D{{"name", mod.Name}}).Decode(&foundMod); err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Unknown error occurred",
				})
			}
		}

		if foundMod.Name != "" {
			mod.CreatedAt = foundMod.CreatedAt
			if _, err := modsCollection.UpdateOne(context.TODO(), bson.D{{"name", foundMod.Name}}, bson.D{{"$set", mod}}); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Error occurred during document update",
				})
			}

			return c.Status(fiber.StatusNoContent).SendString("Document successfully updated")
		}

		mod.CreatedAt = time.Now()
		if _, err := modsCollection.InsertOne(context.TODO(), mod); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Error occurred during document creation",
			})
		}

		return c.Status(fiber.StatusCreated).SendString("Document successfully created")
	})

	app.Delete("/delete/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")

		var deletedDocument bson.M
		if err := modsCollection.FindOneAndDelete(context.TODO(), bson.D{{"name", name}}).Decode(&deletedDocument); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Document does not exist",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Unknown error occurred during deletion",
			})
		}

		return c.Status(fiber.StatusNoContent).SendString("Document successfully removed")
	})

	app.Get("/find/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")

		var foundDocument bson.M
		if err := modsCollection.FindOne(context.TODO(), bson.D{{"name", name}}).Decode(&foundDocument); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Document does not exist",
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Unknown error occurred",
			})
		}

		return c.Status(fiber.StatusOK).JSON(foundDocument)
	})

	if err := app.Listen(":3000"); err != nil {
		panic(err)
	}
}
