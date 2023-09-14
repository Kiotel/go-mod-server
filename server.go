package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

func main() {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://127.0.0.1:27017/"))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	fmt.Print("ping...")
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		panic(err)
	} else {
		fmt.Println("pong")
	}
	modsCollection := client.Database("suxen").Collection("mods")

	type Mod struct {
		Name        string    `json:"name" bson:"name"`
		Description string    `json:"description" bson:"description"`
		CreatedAt   time.Time `bson:"created_at"`
		UpdatedAt   time.Time `bson:"updated_at"`
	}

	app := fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: json.Unmarshal,
	})
	app.Get("/mods", func(c *fiber.Ctx) error {
		var mods []bson.M
		cursor, err := modsCollection.Find(context.TODO(), bson.D{})
		if err != nil {
			return err
		}
		for cursor.Next(context.TODO()) {
			var mod bson.M
			err := cursor.Decode(&mod)
			if err != nil {
				return err
			}
			mods = append(mods, mod)
		}
		return c.Status(200).JSON(mods)
	})
	app.Put("/create", func(c *fiber.Ctx) error {
		mod := new(Mod)
		mod.UpdatedAt = time.Now() //mongo doesn't autofill time(
		err := c.BodyParser(mod)
		if err != nil {
			return c.Status(400).SendString("Error: body fields are missing!👀")
		}

		var foundMod Mod
		err = modsCollection.FindOne(context.TODO(), bson.D{{"name", mod.Name}}).Decode(&foundMod)

		if !errors.Is(err, mongo.ErrNoDocuments) { //if mod exists yet it'll be updated otherwise created
			mod.CreatedAt = foundMod.CreatedAt
			modsCollection.FindOneAndUpdate(context.TODO(), bson.D{{"name", foundMod.Name}}, bson.D{{"$set", mod}})
			return c.Status(204).SendString("Document successfully updated!👍")
		} else {
			mod.CreatedAt = time.Now()
			_, err := modsCollection.InsertOne(context.TODO(), mod)
			if err != nil {
				return c.Status(400).SendString("Error: document not created!🤐")
			}
			return c.Status(201).SendString("Document successfully created!👌")
		}
	})
	app.Delete("/delete/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		var deletedDocument bson.M
		err := modsCollection.FindOneAndDelete(context.TODO(), bson.D{{"name", name}}).Decode(&deletedDocument)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return c.Status(404).SendString("Error: document not exists!👀")
			} else {
				return c.Status(400).SendString("Error: unknown error on delete!🤐")
			}
		}
		return c.Status(204).SendString("Document successfully removed!🙈")
	})
	app.Get("/find/:name", func(c *fiber.Ctx) error {
		name := c.Params("name")
		var foundDocument bson.M
		err := modsCollection.FindOne(context.TODO(), bson.D{{"name", name}}).Decode(&foundDocument)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return c.Status(404).SendString("Error: document not exists!👀")
			} else {
				return c.Status(400).SendString("Error: unknown error!🤐")
			}
		}
		return c.Status(200).JSON(foundDocument)
	})

	err = app.Listen(":3000") // must be in the end
	if err != nil {
		return
	}
}
