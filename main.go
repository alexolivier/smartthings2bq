package main

import (
	"log"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/pubsub"
	"github.com/mitchellh/mapstructure"
	"github.com/namsral/flag"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

var projectPtr string
var datasetPtr string
var tablePtr string
var subscriptionPtr string
var keyfilePtr string

type recording struct {
	ID        string `mapstructure:"id"`
	Timestamp string `mapstructure:"date" bigquery:"timestamp"`
	Device    string `mapstructure:"device"`
	Key       string `mapstructure:"key"`
	Name      string `mapstructure:"name"`
	Desc      string `mapstructure:"desc"`
	Property  string `mapstructure:"property"`
	Value     string `mapstructure:"value"`
	Source    string `mapstructure:"source"`
	Unit      string `mapstructure:"unit"`
	Location  string `mapstructure:"location"`
	Hub       string `mapstructure:"hub"`
	DeviceID  string `mapstructure:"deviceId"`
}

func main() {
	flag.StringVar(&projectPtr, "project", "alex-olivier", "GCP Project")
	flag.StringVar(&datasetPtr, "dataset", "smarthome_dev", "BigQuery Dataset")
	flag.StringVar(&tablePtr, "table", "smartthings", "BigQuery Table")
	flag.StringVar(&subscriptionPtr, "subscription", "smartthings2bq-dev", "Pub/Sub Topic Name")
	flag.StringVar(&keyfilePtr, "keyfile", "default", "Path to keyfile")
	flag.Parse()
	log.Printf("Project: %s", projectPtr)
	log.Printf("Dataset: %s", datasetPtr)
	log.Printf("Table: %s", tablePtr)
	log.Printf("Subscription: %s", subscriptionPtr)
	log.Printf("Keyfile: %s", keyfilePtr)

	ctx := context.Background()

	var bqClient *bigquery.Client
	var pubsubClient *pubsub.Client
	var err error
	if keyfilePtr == "default" {
		bq, e := bigquery.NewClient(ctx, projectPtr)
		bqClient = bq
		ps, e := pubsub.NewClient(ctx, projectPtr)
		pubsubClient = ps
		err = e
	} else {
		bq, e := bigquery.NewClient(ctx, projectPtr, option.WithCredentialsFile(keyfilePtr))
		bqClient = bq
		ps, e := pubsub.NewClient(ctx, projectPtr, option.WithCredentialsFile(keyfilePtr))
		pubsubClient = ps
		err = e
	}
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	} else {
		log.Println("Clients ready")
	}
	uploader := bqClient.Dataset(datasetPtr).Table(tablePtr).Uploader()
	subscription := pubsubClient.Subscription(subscriptionPtr)
	err = subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var record recording
		if err := mapstructure.Decode(msg.Attributes, &record); err != nil {
			log.Printf("could not decode message data: %#v", msg)
			msg.Ack()
			return
		}
		items := []*recording{
			&record,
		}
		if err := uploader.Put(ctx, items); err != nil {
			log.Printf("Failed to insert row: %v", err)
			return
		}
		msg.Ack()
		log.Printf("Inserted %s", msg.ID)
	})
	if err != nil {
		log.Printf("Failed to subscribe: %v", err)
	}
}
