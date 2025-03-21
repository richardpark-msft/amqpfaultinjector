package amqpfaultinjector_test

import (
	"context"
	"crypto/tls"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Azure/amqpfaultinjector"
	"github.com/Azure/amqpfaultinjector/internal/testhelpers"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"github.com/stretchr/testify/require"
)

func TestAMQPProxy(t *testing.T) {
	testData := mustCreateAMQPProxy(t)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	require.NoError(t, err)

	client, err := azservicebus.NewClient(testData.ServiceBusEndpoint, cred, &azservicebus.ClientOptions{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		CustomEndpoint: "127.0.0.1:5671",
		RetryOptions: azservicebus.RetryOptions{
			MaxRetries: -1,
		},
	})
	require.NoError(t, err)

	sender, err := client.NewSender(testData.ServiceBusQueue, nil)
	require.NoError(t, err)

	err = sender.SendMessage(context.Background(), &azservicebus.Message{
		Body: []byte("hello world"),
	}, nil)
	require.NoError(t, err)

	require.NoError(t, sender.Close(context.Background()))

	require.NoError(t, testData.Close())

	testhelpers.ValidateLog(t, testData.JSONLFile+"-1.json")
}

type testAMQPProxy struct {
	*amqpfaultinjector.AMQPProxy
	JSONLFile          string
	ServiceBusEndpoint string
	ServiceBusQueue    string
}

func mustCreateAMQPProxy(t *testing.T) testAMQPProxy {
	dir, err := os.MkdirTemp("", "amqpproxy*")
	require.NoError(t, err)

	t.Logf("Temp folder: %s", dir)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	jsonlFile := path.Join(dir, "amqpproxy-traffic")

	amqpProxy, err := amqpfaultinjector.NewAMQPProxy(
		"localhost:5671",
		serviceBusEndpoint,
		&amqpfaultinjector.AMQPProxyOptions{
			BaseJSONName: jsonlFile,
			CertDir:      dir,
		})
	require.NoError(t, err)

	go func() {
		require.NoError(t, amqpProxy.ListenAndServe())
	}()

	time.Sleep(5 * time.Second)

	return testAMQPProxy{
		AMQPProxy:          amqpProxy,
		JSONLFile:          jsonlFile,
		ServiceBusEndpoint: serviceBusEndpoint,
		ServiceBusQueue:    serviceBusQueue,
	}
}
