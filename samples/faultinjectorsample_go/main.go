package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/joho/godotenv"
)

func main() {
	err := func() error {
		if len(os.Args) != 2 {
			return fmt.Errorf("no command specified\nUsage: <program> [servicebus|eventhubs]\n")
		}

		switch os.Args[1] {
		case "servicebus":
			if err := runServiceBusSample(); err != nil {
				return fmt.Errorf("failed to run servicebus demo: %w", err)
			}
		case "eventhubs":
			if err := runEventHubsSample(); err != nil {
				return fmt.Errorf("failed to run eventhubs demo: %w", err)
			}
		default:
			return fmt.Errorf("unknown command %q\nUsage: <program> [servicebus|eventhubs]", os.Args[1])
		}

		fmt.Printf("Sample completed successfully\n")
		return nil
	}()

	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

func runEventHubsSample() error {
	values, err := mustGetEnvVars(
		"EVENTHUBS_FULLY_QUALIFIED_NAMESPACE",
		"EVENTHUBS_HUB_NAME")

	if err != nil {
		return err
	}

	endpoint, hub := values[0], values[1]

	dac, err := azidentity.NewDefaultAzureCredential(nil)

	if err != nil {
		return err
	}

	client, err := azeventhubs.NewProducerClient(endpoint, hub, dac, &azeventhubs.ProducerClientOptions{
		CustomEndpoint: "localhost:5671",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	if err != nil {
		return err
	}

	defer func() {
		err = client.Close(context.Background())

		if err != nil {
			fmt.Printf("Failed when closing event hubs client: %s\n", err)
		}
	}()

	batch, err := client.NewEventDataBatch(context.Background(), nil)

	if err != nil {
		return err
	}

	err = batch.AddEventData(&azeventhubs.EventData{
		Body: []byte("hello world"),
	}, nil)

	if err != nil {
		return err
	}

	return client.SendEventDataBatch(context.Background(), batch, nil)
}

func runServiceBusSample() error {
	values, err := mustGetEnvVars(
		"SERVICEBUS_FULLY_QUALIFIED_NAMESPACE",
		"SERVICEBUS_QUEUE_NAME")

	if err != nil {
		return err
	}

	endpoint, queue := values[0], values[1]

	dac, err := azidentity.NewDefaultAzureCredential(nil)

	if err != nil {
		return err
	}

	// TODO: I'm reversed from what I think other people are doing - the endpoint is the "real"
	// endpoint, and the hostname is supposed to be localhost:5671.
	client, err := azservicebus.NewClient(endpoint, dac, &azservicebus.ClientOptions{
		CustomEndpoint: "localhost:5671",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	if err != nil {
		return err
	}

	sender, err := client.NewSender(queue, nil)

	if err != nil {
		return err
	}

	err = sender.SendMessage(context.Background(), &azservicebus.Message{
		Body: []byte("hello world!"),
	}, nil)

	return err
}

func mustGetEnvVars(vars ...string) ([]string, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %s", err)
	}

	var missing []string
	var values []string

	for _, v := range vars {
		val := os.Getenv(v)

		if val == "" {
			missing = append(missing, v)
			continue
		}

		values = append(values, val)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing environment variables: %#v", missing)
	}

	return values, nil
}
