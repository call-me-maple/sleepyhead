package function

import (
	"context"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	_ "github.com/cloudevents/sdk-go/v2"
	"log"
)

func init() {
	funcframework.RegisterEventFunction("/", HelloPubSub)
}

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Data []byte `json:"data"`
}

// HelloPubSub consumes a Pub/Sub message.
func HelloPubSub(ctx context.Context, m PubSubMessage) error {
	name := string(m.Data) // Automatically decoded from base64.
	if name == "" {
		name = "there"
	}
	log.Printf("Hello, %s!", name)
	return nil
}
