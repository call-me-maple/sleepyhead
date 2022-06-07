package sleepyhead

import (
	computec "cloud.google.com/go/compute/apiv1"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/bwmarrin/discordgo"
	_ "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/compute/v1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
	"strings"
	"time"
)

var (
	project          = os.Getenv("PROJECT")
	zone             = os.Getenv("ZONE")
	instanceName     = os.Getenv("INSTANCE")
	instanceTemplate = os.Getenv("INSTANCE_TEMPLATE")
	botToken         = os.Getenv("BOT_TOKEN")
	computeService   *compute.Service
)

func init() {
	if project == "" {
		log.Fatal("$PROJECT not set")
	}
	if zone == "" {
		log.Fatal("$ZONE not set")
	}
	if instanceName == "" {
		log.Fatal("$INSTANCE not set")
	}
	if botToken == "" {
		log.Fatal("$BOT_TOKEN not set")
	}

	funcframework.RegisterEventFunction("/", HiSleepyHead)
	ctx := context.Background()
	cs, err := compute.NewService(ctx)
	computeService = cs
	if err != nil {
		log.Fatal(err)
	}
}

type PubSubMessage struct {
	Data []byte `json:"data"`
}

func HiSleepyHead(ctx context.Context, m PubSubMessage) (err error) {
	return main()
}

func main() (err error) {
	dg, err := discordgo.New(fmt.Sprintf("Bot %v", strings.TrimSpace(botToken)))
	if err != nil {
		log.Println("error creating Discord session,", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}
	defer dg.Close()
	gm, channelID := isGoodMorning(dg)
	if !gm {
		log.Println("not good morning so nothing to check")
		return
	}

	instance, err := findSleepyhead()
	if err != nil {
		log.Println(err)
		instance, err = wakeUpSleepyhead()
		if err != nil {
			log.Println(err)
			return
		}
	} else if instance.Status == "RUNNING" {
		log.Println("sleepyhead awake already")
		dg.ChannelMessageSend(channelID, "sleepyhead awake already...")
		return
	}

	switch instance.Status {
	case "RUNNING", "PROVISIONING":
	case "REPAIRING", "STAGING", "STOPPED", "STOPPING", "SUSPENDED", "SUSPENDING", "TERMINATED":
		err = removeSleepyhead()
		if err != nil {
			log.Println(err)
		}
		instance, err = wakeUpSleepyhead()
		if err != nil {
			log.Println(err)
			return
		}
	default:
		log.Printf("unknown instance status: %v\n", instance.Status)
		return
	}
	log.Println("sleepyhead waking up 🌅")
	dg.ChannelMessageSend(channelID, "sleepyhead waking up 🌅")
	return
}

func isGoodMorning(dg *discordgo.Session) (gm bool, channelID string) {
	for _, guild := range dg.State.Guilds {
		channels, err := dg.GuildChannels(guild.ID)
		if err != nil {
			continue
		}
		for _, channel := range channels {
			messages, err := dg.ChannelMessages(channel.ID, 100, "", "", channel.LastMessageID)
			if err != nil {
				continue
			}
			for _, message := range messages {
				// Check for bot mention
				for _, mention := range message.Mentions {
					if mention.ID != dg.State.User.ID {
						continue
					}
					if time.Since(message.Timestamp) <= time.Minute*3 && strings.Contains(message.Content, "gm") {
						gm = true
						channelID = channel.ID
						dg.ChannelMessageSend(channelID, "checking in on sleepyhead")
						return
					}
				}
			}
		}
	}
	return
}

func removeSleepyhead() (err error) {
	ctx := context.Background()

	_, err = computeService.Instances.Delete(project, zone, instanceName).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("removed sleepyhead")
	return
}

func findSleepyhead() (instance *compute.Instance, err error) {
	ctx := context.Background()
	instance, err = computeService.Instances.Get(project, zone, instanceName).Context(ctx).Do()

	req := computeService.Instances.List(project, zone)
	instances := make([]*compute.Instance, 0)
	if err = req.Filter(fmt.Sprintf("name=%v", instanceName)).Pages(ctx, func(page *compute.InstanceList) error {
		for _, instance = range page.Items {
			instances = append(instances, instance)
		}
		return nil
	}); err != nil {
		log.Println(err)
	}

	switch len(instances) {
	case 0:
		err = errors.Errorf("can't find the sleepyhead")
	case 1:
		log.Println("found the sleepyhead")
		instance = instances[0]
	default:
		err = errors.Errorf("found too many sleepyheads")
	}
	return
}

func wakeUpSleepyhead() (instance *compute.Instance, err error) {
	ctx := context.Background()

	instancesClient, err := computec.NewInstancesRESTClient(ctx)
	if err != nil {
		err = errors.Wrap(err, "NewInstancesRESTClient:")
		return
	}
	defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
		Project: project,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name: proto.String(instanceName),
		},
		SourceInstanceTemplate: &instanceTemplate,
	}

	op, err := instancesClient.Insert(ctx, req)
	if err != nil {
		err = errors.Wrap(err, "unable to create instance:")
		return
	}

	if err = op.Wait(ctx); err != nil {
		err = errors.Wrap(err, "unable to wait for the operation: %v")
		return
	}

	log.Println("woke up sleepyhead")

	return findSleepyhead()
}
